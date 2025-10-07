package services

import (
	"fmt"
	"strings"
)

type PromptBuilder struct{}

func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildCVEvaluationPrompt creates prompt for CV evaluation
func (pb *PromptBuilder) BuildCVEvaluationPrompt(cvText, jobDescription, scoringRubric, jobTitle string) string {
	return fmt.Sprintf(`You are an expert HR recruiter evaluating a candidate's CV for a %s position.

JOB DESCRIPTION:
%s

SCORING RUBRIC:
%s

CANDIDATE CV:
%s

Your task is to evaluate the candidate's CV against the job description using the scoring rubric provided.

Evaluate the following parameters (1-5 scale):
1. Technical Skills Match (Weight: 40%%) - Alignment with job requirements (backend, databases, APIs, cloud, AI/LLM)
2. Experience Level (Weight: 25%%) - Years of experience and project complexity
3. Relevant Achievements (Weight: 20%%) - Impact of past work (scaling, performance, adoption)
4. Cultural/Collaboration Fit (Weight: 15%%) - Communication, learning mindset, teamwork/leadership

Return your response in the following JSON format:
{
  "technical_skills_score": <1-5>,
  "experience_level_score": <1-5>,
  "achievements_score": <1-5>,
  "cultural_fit_score": <1-5>,
  "weighted_average": <calculated weighted average>,
  "match_rate": <weighted_average * 0.2, as decimal 0-1>,
  "feedback": "<detailed feedback 3-5 sentences explaining strengths and gaps>"
}

Be objective and thorough. Provide specific examples from the CV to justify your scores.`,
		jobTitle, jobDescription, scoringRubric, cvText)
}

// BuildProjectEvaluationPrompt creates prompt for project report evaluation
func (pb *PromptBuilder) BuildProjectEvaluationPrompt(projectText, caseStudyBrief, scoringRubric string) string {
	return fmt.Sprintf(`You are an expert technical evaluator assessing a candidate's project report for a backend developer take-home assignment.

CASE STUDY BRIEF (Requirements):
%s

SCORING RUBRIC:
%s

CANDIDATE'S PROJECT REPORT:
%s

Your task is to evaluate the candidate's project report against the case study requirements using the scoring rubric.

Evaluate the following parameters (1-5 scale):
1. Correctness (Weight: 30%%) - Implements prompt design, LLM chaining, RAG context injection
2. Code Quality & Structure (Weight: 25%%) - Clean, modular, reusable, tested
3. Resilience & Error Handling (Weight: 20%%) - Handles long jobs, retries, randomness, API failures
4. Documentation & Explanation (Weight: 15%%) - README clarity, setup instructions, trade-off explanations
5. Creativity/Bonus (Weight: 10%%) - Extra features beyond requirements

Return your response in the following JSON format:
{
  "correctness_score": <1-5>,
  "code_quality_score": <1-5>,
  "resilience_score": <1-5>,
  "documentation_score": <1-5>,
  "creativity_score": <1-5>,
  "weighted_average": <calculated weighted average>,
  "project_score": <weighted_average as decimal>,
  "feedback": "<detailed feedback 3-5 sentences explaining what was done well and what could be improved>"
}

Be thorough and specific. Reference actual implementation details from the report.`,
		caseStudyBrief, scoringRubric, projectText)
}

// BuildFinalSummaryPrompt creates prompt for overall summary
func (pb *PromptBuilder) BuildFinalSummaryPrompt(cvFeedback, projectFeedback string, cvMatchRate, projectScore float64, jobTitle string) string {
	return fmt.Sprintf(`You are an expert technical hiring manager making a final assessment of a candidate for a %s position.

CV EVALUATION RESULTS:
- Match Rate: %.2f (out of 1.0)
- Feedback: %s

PROJECT EVALUATION RESULTS:
- Project Score: %.2f (out of 5.0)
- Feedback: %s

Based on both evaluations, provide a concise overall summary (3-5 sentences) that includes:
1. Overall strengths of the candidate
2. Key gaps or areas for improvement
3. Final recommendation (Strong Hire / Hire / Maybe / No Hire)

Return ONLY the summary text, no JSON format needed. Be direct and actionable.`,
		jobTitle, cvMatchRate, cvFeedback, projectScore, projectFeedback)
}

// BuildRetrievalQuery creates query for RAG retrieval
func (pb *PromptBuilder) BuildRetrievalQuery(queryType, context string) string {
	switch queryType {
	case "job_description":
		return fmt.Sprintf("Job requirements and qualifications for %s", context)
	case "case_study":
		return "Project requirements, technical specifications, and evaluation criteria"
	case "cv_rubric":
		return "CV evaluation criteria and scoring guidelines"
	case "project_rubric":
		return "Project evaluation criteria and scoring guidelines"
	default:
		return context
	}
}

// Helper to clean and format context from RAG results
func FormatRAGContext(results []SearchResult) string {
	if len(results) == 0 {
		return "No relevant context found."
	}

	var parts []string
	for i, result := range results {
		parts = append(parts, fmt.Sprintf("--- Context %d (Score: %.2f) ---\n%s",
			i+1, result.Score, strings.TrimSpace(result.Text)))
	}

	return strings.Join(parts, "\n\n")
}
