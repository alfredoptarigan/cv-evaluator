package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"

	"alfredoptarigan/cv-evaluator/internal/models"
	"alfredoptarigan/cv-evaluator/internal/repositories"
)

type EvaluatorService interface {
	EvaluateCandidate(ctx context.Context, evalID uuid.UUID) error
}

type evaluatorService struct {
	evalRepo      repositories.EvaluationRepository
	docRepo       repositories.DocumentRepository
	geminiService GeminiService
	qdrantService QdrantService
	pdfParser     PDFParserService
	promptBuilder *PromptBuilder
	maxRetries    int
}

func NewEvaluatorService(
	evalRepo repositories.EvaluationRepository,
	docRepo repositories.DocumentRepository,
	geminiService GeminiService,
	qdrantService QdrantService,
	pdfParser PDFParserService,
	maxRetries int,
) EvaluatorService {
	return &evaluatorService{
		evalRepo:      evalRepo,
		docRepo:       docRepo,
		geminiService: geminiService,
		qdrantService: qdrantService,
		pdfParser:     pdfParser,
		promptBuilder: NewPromptBuilder(),
		maxRetries:    maxRetries,
	}
}

type CVEvaluationResult struct {
	TechnicalSkillsScore float64 `json:"technical_skills_score"`
	ExperienceLevelScore float64 `json:"experience_level_score"`
	AchievementsScore    float64 `json:"achievements_score"`
	CulturalFitScore     float64 `json:"cultural_fit_score"`
	WeightedAverage      float64 `json:"weighted_average"`
	MatchRate            float64 `json:"match_rate"`
	Feedback             string  `json:"feedback"`
}

type ProjectEvaluationResult struct {
	CorrectnessScore   float64 `json:"correctness_score"`
	CodeQualityScore   float64 `json:"code_quality_score"`
	ResilienceScore    float64 `json:"resilience_score"`
	DocumentationScore float64 `json:"documentation_score"`
	CreativityScore    float64 `json:"creativity_score"`
	WeightedAverage    float64 `json:"weighted_average"`
	ProjectScore       float64 `json:"project_score"`
	Feedback           string  `json:"feedback"`
}

func (e *evaluatorService) EvaluateCandidate(ctx context.Context, evalID uuid.UUID) error {
	// Update status to processing
	if err := e.evalRepo.UpdateStatus(evalID, models.StatusProcessing); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	log.Printf("🔄 Starting evaluation for job ID: %s\n", evalID)

	// Get evaluation details
	evaluation, err := e.evalRepo.FindByID(evalID)
	if err != nil {
		e.evalRepo.UpdateError(evalID, err.Error())
		return fmt.Errorf("failed to get evaluation: %w", err)
	}

	// Get documents
	cvDoc, err := e.docRepo.FindByID(evaluation.CVDocumentID)
	if err != nil {
		e.evalRepo.UpdateError(evalID, fmt.Sprintf("CV document not found: %v", err))
		return fmt.Errorf("failed to get CV document: %w", err)
	}

	projectDoc, err := e.docRepo.FindByID(evaluation.ProjectDocumentID)
	if err != nil {
		e.evalRepo.UpdateError(evalID, fmt.Sprintf("Project document not found: %v", err))
		return fmt.Errorf("failed to get project document: %w", err)
	}

	// Step 1: Parse PDFs
	log.Println("📄 Parsing CV...")
	cvContent, err := e.pdfParser.ExtractTextWithMetaData(cvDoc.FilePath)
	if err != nil {
		e.evalRepo.UpdateError(evalID, fmt.Sprintf("Failed to parse CV: %v", err))
		return fmt.Errorf("failed to parse CV: %w", err)
	}

	projectContent, err := e.pdfParser.ExtractTextWithMetaData(projectDoc.FilePath)
	if err != nil {
		e.evalRepo.UpdateError(evalID, fmt.Sprintf("Failed to parse project report: %v", err))
		return fmt.Errorf("failed to parse project report: %w", err)
	}

	// Step 2: Retrieve relevant context from Qdrant (RAG)
	log.Println("🔍 Retrieving relevant context for CV evaluation...")
	cvContext, err := e.retrieveContext(ctx, cvContent.Text, []string{"job_description", "cv_rubric"})
	if err != nil {
		log.Printf("⚠️  Warning: Failed to retrieve CV context: %v\n", err)
		cvContext = ""
	}

	log.Println("🔍 Retrieving relevant context for Project evaluation...")
	projectContext, err := e.retrieveContext(ctx, projectContent.Text, []string{"case_study", "project_rubric"})
	if err != nil {
		log.Printf("⚠️  Warning: Failed to retrieve project context: %v\n", err)
		projectContext = ""
	}

	// Step 3: Evaluate CV
	log.Println("🤖 Evaluating CV with LLM...")
	cvResult, err := e.evaluateCV(ctx, cvContent.Text, cvContext, evaluation.JobTitle)
	if err != nil {
		e.evalRepo.UpdateError(evalID, fmt.Sprintf("Failed to evaluate CV: %v", err))
		return fmt.Errorf("failed to evaluate CV: %w", err)
	}

	// Step 4: Evaluate Project
	log.Println("🤖 Evaluating Project Report with LLM...")
	projectResult, err := e.evaluateProject(ctx, projectContent.Text, projectContext)
	if err != nil {
		e.evalRepo.UpdateError(evalID, fmt.Sprintf("Failed to evaluate project: %v", err))
		return fmt.Errorf("failed to evaluate project: %w", err)
	}

	// Step 5: Generate Overall Summary
	log.Println("🤖 Generating overall summary...")
	overallSummary, err := e.generateSummary(ctx, cvResult, projectResult, evaluation.JobTitle)
	if err != nil {
		e.evalRepo.UpdateError(evalID, fmt.Sprintf("Failed to generate summary: %v", err))
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Step 6: Save results
	log.Println("💾 Saving evaluation results...")
	updateData := &repositories.EvaluationUpdateData{
		CVMatchRate:     &cvResult.MatchRate,
		CVFeedback:      &cvResult.Feedback,
		ProjectScore:    &projectResult.ProjectScore,
		ProjectFeedback: &projectResult.Feedback,
		OverallSummary:  &overallSummary,
	}

	if err := e.evalRepo.UpdateResult(evalID, updateData); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	log.Printf("✅ Evaluation completed successfully for job ID: %s\n", evalID)
	return nil
}

func (e *evaluatorService) retrieveContext(ctx context.Context, queryText string, docTypes []string) (string, error) {
	// Generate embedding for query
	embedding, err := e.geminiService.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return "", fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search for each doc type
	var allResults []SearchResult
	for _, docType := range docTypes {
		results, err := e.qdrantService.SearchSimilar(ctx, embedding, docType, 3)
		if err != nil {
			log.Printf("⚠️  Failed to search for %s: %v\n", docType, err)
			continue
		}
		allResults = append(allResults, results...)
	}

	return FormatRAGContext(allResults), nil
}

func (e *evaluatorService) evaluateCV(ctx context.Context, cvText, context, jobTitle string) (*CVEvaluationResult, error) {
	prompt := e.promptBuilder.BuildCVEvaluationPrompt(cvText, context, "", jobTitle)

	// Log prompt length for debugging
	log.Printf("📝 CV Evaluation prompt length: %d characters", len(prompt))

	// Generate with retry
	response, err := e.geminiService.GenerateTextWithRetry(ctx, prompt, 0.3, e.maxRetries)
	if err != nil {
		log.Printf("❌ CV Evaluation failed: %v", err)
		return nil, fmt.Errorf("failed to generate CV evaluation: %w", err)
	}

	// Log response for debugging
	log.Printf("✅ CV Evaluation response received: %d characters", len(response))

	// Check for empty response
	if response == "" {
		log.Println("⚠️ Empty response received from Gemini API")
		// Fallback to default evaluation result with error message
		return &CVEvaluationResult{
			TechnicalSkillsScore: 0,
			ExperienceLevelScore: 0,
			AchievementsScore:    0,
			CulturalFitScore:     0,
			WeightedAverage:      0,
			MatchRate:            0,
			Feedback:             "Failed to evaluate CV due to API response issues. Please try again later.",
		}, nil
	}

	// Parse JSON response
	var result CVEvaluationResult
	if err := e.parseJSONResponse(response, &result); err != nil {
		log.Printf("❌ Failed to parse CV evaluation response: %v", err)
		return nil, fmt.Errorf("failed to parse CV evaluation response: %w", err)
	}

	return &result, nil
}

func (e *evaluatorService) evaluateProject(ctx context.Context, projectText, context string) (*ProjectEvaluationResult, error) {
	prompt := e.promptBuilder.BuildProjectEvaluationPrompt(projectText, context, "")

	// Log prompt length for debugging
	log.Printf("📝 Project Evaluation prompt length: %d characters", len(prompt))

	// Generate with retry
	response, err := e.geminiService.GenerateTextWithRetry(ctx, prompt, 0.3, e.maxRetries)
	if err != nil {
		log.Printf("❌ Project Evaluation failed: %v", err)
		return nil, fmt.Errorf("failed to generate project evaluation: %w", err)
	}

	// Log response for debugging
	log.Printf("✅ Project Evaluation response received: %d characters", len(response))

	// Check for empty response
	if response == "" {
		log.Println("⚠️ Empty response received from Gemini API")
		// Fallback to default evaluation result with error message
		return &ProjectEvaluationResult{
			CorrectnessScore:   0,
			CodeQualityScore:   0,
			ResilienceScore:    0,
			DocumentationScore: 0,
			CreativityScore:    0,
			WeightedAverage:    0,
			ProjectScore:       0,
			Feedback:           "Failed to evaluate project due to API response issues. Please try again later.",
		}, nil
	}

	// Parse JSON response
	var result ProjectEvaluationResult
	if err := e.parseJSONResponse(response, &result); err != nil {
		log.Printf("❌ Failed to parse project evaluation response: %v", err)
		return nil, fmt.Errorf("failed to parse project evaluation response: %w", err)
	}

	return &result, nil
}

func (e *evaluatorService) generateSummary(ctx context.Context, cvResult *CVEvaluationResult, projectResult *ProjectEvaluationResult, jobTitle string) (string, error) {
	prompt := e.promptBuilder.BuildFinalSummaryPrompt(
		cvResult.Feedback,
		projectResult.Feedback,
		cvResult.MatchRate,
		projectResult.ProjectScore,
		jobTitle,
	)

	// Generate with retry
	summary, err := e.geminiService.GenerateTextWithRetry(ctx, prompt, 0.5, e.maxRetries)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return strings.TrimSpace(summary), nil
}

func (e *evaluatorService) parseJSONResponse(response string, target interface{}) error {
	// Try to extract JSON from response (LLM might wrap it in markdown)
	jsonStr := extractJSON(response)

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w\nResponse: %s", err, response)
	}

	return nil
}

// extractJSON tries to extract JSON from text that might contain markdown or other formatting
func extractJSON(text string) string {
	// Remove markdown code blocks
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")

	// Find JSON object or array boundaries
	startObj := strings.Index(text, "{")
	startArr := strings.Index(text, "[")
	endObj := strings.LastIndex(text, "}")
	endArr := strings.LastIndex(text, "]")

	// Determine if we have an object or array
	if startObj != -1 && endObj != -1 && endObj > startObj {
		// We have a JSON object
		return text[startObj : endObj+1]
	} else if startArr != -1 && endArr != -1 && endArr > startArr {
		// We have a JSON array
		return text[startArr : endArr+1]
	}

	return text
}
