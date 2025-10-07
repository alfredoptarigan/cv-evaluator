package services

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

type QdrantService interface {
	InitCollection() error
	UpsertDocument(ctx context.Context, docID string, docType string, text string, embedding []float32) error
	SearchSimilar(ctx context.Context, queryEmbedding []float32, docType string, limit int) ([]SearchResult, error)
	DeleteDocument(ctx context.Context, docID string) error
}

type SearchResult struct {
	ID       string
	Score    float32
	Text     string
	DocType  string
	Metadata map[string]interface{}
}

type qdrantService struct {
	client         *qdrant.Client
	collectionName string
	vectorSize     uint64
}

func NewQdrantService(urlStr, apiKey, collectionName string) (QdrantService, error) {
	// Parse URL to extract host, port, and TLS usage
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid Qdrant URL: %w", err)
	}

	host := parsed.Hostname()
	useTLS := parsed.Scheme == "https"

	// For gRPC client, use port 6334 by default (gRPC port)
	port := 6334
	if p := parsed.Port(); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
		UseTLS: useTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	return &qdrantService{
		client:         client,
		collectionName: collectionName,
		vectorSize:     768, // OpenAI embedding size
	}, nil
}

// InitCollection implements QdrantService.
func (q *qdrantService) InitCollection() error {
	ctx := context.Background()

	// Check if collection exists
	exists, err := q.client.CollectionExists(ctx, q.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}

	if exists {
		log.Println("✅ Collection already exists")
		return nil
	}

	// Create collection
	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: q.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     q.vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	log.Printf("✅ Qdrant collection '%s' created successfully\n", q.collectionName)
	return nil
}

// UpsertDocument implements QdrantService.
func (q *qdrantService) UpsertDocument(ctx context.Context, docID string, docType string, text string, embedding []float32) error {
	pointID := uuid.New()

	point := &qdrant.PointStruct{
		Id:      qdrant.NewIDNum(uint64(pointID.ID())),
		Vectors: qdrant.NewVectors(embedding...),
		Payload: qdrant.NewValueMap(map[string]interface{}{
			"doc_id":   docID,
			"doc_type": docType,
			"text":     text,
		}),
	}

	// Upsert point
	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collectionName,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	return nil
}

// SearchSimilar implements QdrantService.
func (q *qdrantService) SearchSimilar(ctx context.Context, queryEmbedding []float32, docType string, limit int) ([]SearchResult, error) {
	var filter *qdrant.Filter
	if docType != "" {
		filter = &qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("doc_type", docType),
			},
		}
	}

	searchResult, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.collectionName,
		Query:          qdrant.NewQuery(queryEmbedding...),
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Convert results
	var results []SearchResult
	for _, point := range searchResult {
		payload := point.Payload

		result := SearchResult{
			Score:    point.Score,
			Metadata: make(map[string]interface{}),
		}

		// Extract payload
		if docID, ok := payload["doc_id"]; ok {
			if val, ok := docID.GetKind().(*qdrant.Value_StringValue); ok {
				result.ID = val.StringValue
			}
		}

		if text, ok := payload["text"]; ok {
			if val, ok := text.GetKind().(*qdrant.Value_StringValue); ok {
				result.Text = val.StringValue
			}
		}

		if dtype, ok := payload["doc_type"]; ok {
			if val, ok := dtype.GetKind().(*qdrant.Value_StringValue); ok {
				result.DocType = val.StringValue
			}
		}

		// Store all metadata
		for key, value := range payload {
			result.Metadata[key] = value
		}

		results = append(results, result)
	}

	return results, nil
}

// DeleteDocument implements QdrantService.
func (q *qdrantService) DeleteDocument(ctx context.Context, docID string) error {
	// Delete by filter
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatch("doc_id", docID),
		},
	}

	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: q.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: filter,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}
