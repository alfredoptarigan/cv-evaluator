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
	ID                uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id" column:"id"`
	JobTitle          string           `gorm:"type:text" json:"job_title" column:"job_title"`
	CVDocumentID      uuid.UUID        `gorm:"type:uuid;not null" json:"cv_document_id" column:"cv_document_id"`
	ProjectDocumentID uuid.UUID        `gorm:"type:uuid;not null" json:"project_document_id" column:"project_document_id"`
	Status            EvaluationStatus `gorm:"not null;default:'queued'" json:"status" column:"status"`
	CVMatchRate       float64          `gorm:"column:cv_match_rate" json:"cv_match_rate"`
	CVFeedback        string           `gorm:"type:text" json:"cv_feedback,omitempty" column:"cv_feedback"`
	ProjectScore      float64          `gorm:"column:project_score" json:"project_score,omitempty"`
	ProjectFeedback   string           `gorm:"type:text" json:"project_feedback,omitempty" column:"project_feedback"`
	OverallSummary    string           `gorm:"type:text" json:"overall_summary,omitempty" column:"overall_summary"`
	ErrorMessage      string           `gorm:"type:text" json:"error_message,omitempty" column:"error_message"`
	CreatedAt         time.Time        `gorm:"default:CURRENT_TIMESTAMP" json:"created_at" column:"created_at"`
	UpdatedAt         time.Time        `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at" column:"updated_at"`

	// Relations
	CVDocument      Document `gorm:"foreignKey:CVDocumentID" json:"-"`
	ProjectDocument Document `gorm:"foreignKey:ProjectDocumentID" json:"-"`
}

func (Evaluation) TableName() string {
	return "evaluations"
}
