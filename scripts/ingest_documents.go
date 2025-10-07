package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"alfredoptarigan/cv-evaluator/internal/config"
	"alfredoptarigan/cv-evaluator/internal/services"
)

func main() {
	log.Println("🚀 Starting document ingestion...")

	// Load configuration
	cfg := config.Load()

	// Initialize services
	geminiService, err := services.NewGeminiService(cfg.Gemini.APIKey)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Gemini: %v", err)
	}

	qdrantService, err := services.NewQdrantService(
		cfg.Qdrant.URL,
		cfg.Qdrant.APIKey,
		cfg.Qdrant.Collection,
	)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Qdrant: %v", err)
	}

	if err := qdrantService.InitCollection(); err != nil {
		log.Fatalf("❌ Failed to initialize collection: %v", err)
	}

	pdfParser := services.NewPDFParserService()
	chunker := services.NewTextChunker()

	ctx := context.Background()

	documents := []struct {
		Path    string
		DocType string
		Name    string
	}{
		{
			Path:    "./reference_docs/Job_Description.pdf",
			DocType: "job_description",
			Name:    "Job Description - Product Engineer (Backend)",
		},
		{
			Path:    "./reference_docs/case_study_brief.pdf",
			DocType: "case_study",
			Name:    "Case Study Brief",
		},
		{
			Path:    "./reference_docs/scoring_rubric.pdf",
			DocType: "cv_rubric",
			Name:    "CV Scoring Rubric",
		},
		{
			Path:    "./reference_docs/Study_Case_Submission.pdf",
			DocType: "case_study",
			Name:    "Study Case Submission",
		},
	}

	successCount := 0
	failCount := 0

	for _, doc := range documents {
		log.Printf("\n📄 Processing: %s", doc.Name)
		log.Printf("   Path: %s", doc.Path)
		log.Printf("   Type: %s", doc.DocType)

		// Check if file exists
		if _, err := os.Stat(doc.Path); os.IsNotExist(err) {
			log.Printf("   ⚠️  File not found, skipping...")
			failCount++
			continue
		}

		// Extract text from PDF
		log.Printf("   📖 Extracting text...")
		content, err := pdfParser.ExtractTextWithMetaData(doc.Path)
		if err != nil {
			log.Printf("   ❌ Failed to extract text: %v", err)
			failCount++
			continue
		}

		log.Printf("   ✅ Extracted %d pages, %d characters", content.PageCount, len(content.Text))

		// Chunk the text
		log.Printf("   ✂️  Chunking text...")
		chunks := chunker.ChunkText(content.Text, 1000, 200)
		log.Printf("   ✅ Created %d chunks", len(chunks))

		// Embed and store each chunk
		log.Printf("   🔄 Embedding and storing chunks...")
		for i, chunk := range chunks {
			// Generate embedding
			embedding, err := geminiService.GenerateEmbedding(ctx, chunk)
			if err != nil {
				log.Printf("   ❌ Failed to generate embedding for chunk %d: %v", i+1, err)
				continue
			}

			// Create document ID
			docID := fmt.Sprintf("%s_chunk_%d", doc.DocType, i)

			// Store in Qdrant
			err = qdrantService.UpsertDocument(ctx, docID, doc.DocType, chunk, embedding)
			if err != nil {
				log.Printf("   ❌ Failed to store chunk %d: %v", i+1, err)
				continue
			}

			if (i+1)%5 == 0 || i == len(chunks)-1 {
				log.Printf("   📊 Progress: %d/%d chunks stored", i+1, len(chunks))
			}
		}

		log.Printf("   ✅ Successfully ingested %s", doc.Name)
		successCount++
	}

	// Summary
	log.Println("\n" + strings.Repeat("=", 60))
	log.Printf("📊 Ingestion Summary:")
	log.Printf("   ✅ Successful: %d documents", successCount)
	log.Printf("   ❌ Failed: %d documents", failCount)
	log.Println(strings.Repeat("=", 60))

	if failCount > 0 {
		log.Println("⚠️  Some documents failed to ingest. Please check the logs above.")
		os.Exit(1)
	}

	log.Println("✅ All documents ingested successfully!")
}
