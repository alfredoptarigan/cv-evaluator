package models

import (
	"time"

	"github.com/google/uuid"
)

type EvaluationStatus string

const (
	StatusQueued     EvaluationStatus = "queued"
	StatusProcessing EvaluationStatus = "processing"
	StatusCompleted  EvaluationStatus = "completed"
	StatusFailed     EvaluationStatus = "failed"
)

type Evaluation struct {
	ID                uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	JobTitle          string           `gorm:"type:text" json:"job_title"`
	CVDocumentID      uuid.UUID        `gorm:"type:uuid;not null" json:"cv_document_id"`
	ProjectDocumentID uuid.UUID        `gorm:"type:uuid;not null" json:"project_document_id"`
	Status            EvaluationStatus `gorm:"not null;default:'queued'" json:"status"`
	CVMatchRate       *float64         `gorm:"type:decimal(3,2)" json:"cv_match_rate,omitempty"`
	CVFeedback        *string          `gorm:"type:text" json:"cv_feedback,omitempty"`
	ProjectScore      *float64         `gorm:"type:decimal(3,2)" json:"project_score,omitempty"`
	ProjectFeedback   *string          `gorm:"type:text" json:"project_feedback,omitempty"`
	OverallSummary    *string          `gorm:"type:text" json:"overall_summary,omitempty"`
	ErrorMessage      *string          `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt         time.Time        `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt         time.Time        `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relations
	CVDocument      Document `gorm:"foreignKey:CVDocumentID" json:"-"`
	ProjectDocument Document `gorm:"foreignKey:ProjectDocumentID" json:"-"`
}

func (Evaluation) TableName() string {
	return "evaluations"
}
