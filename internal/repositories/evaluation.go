package repositories

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"alfredoptarigan/cv-evaluator/internal/models"
)

type EvaluationRepository interface {
	Create(eval *models.Evaluation) error
	FindByID(id uuid.UUID) (*models.Evaluation, error)
	UpdateStatus(id uuid.UUID, status models.EvaluationStatus) error
	UpdateResult(id uuid.UUID, result *EvaluationUpdateData) error
	UpdateError(id uuid.UUID, errorMsg string) error
	FindPendingJobs(limit int) ([]models.Evaluation, error)
}

type EvaluationUpdateData struct {
	CVMatchRate     *float64
	CVFeedback      *string
	ProjectScore    *float64
	ProjectFeedback *string
	OverallSummary  *string
}

type evaluationRepository struct {
	db *gorm.DB
}

func NewEvaluationRepository(db *gorm.DB) EvaluationRepository {
	return &evaluationRepository{db: db}
}

func (r *evaluationRepository) Create(eval *models.Evaluation) error {
	if err := r.db.Create(eval).Error; err != nil {
		return fmt.Errorf("failed to create evaluation: %w", err)
	}
	return nil
}

func (r *evaluationRepository) FindByID(id uuid.UUID) (*models.Evaluation, error) {
	var eval models.Evaluation
	if err := r.db.Where("id = ?", id).First(&eval).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("evaluation not found")
		}
		return nil, fmt.Errorf("failed to find evaluation: %w", err)
	}
	return &eval, nil
}

func (r *evaluationRepository) UpdateStatus(id uuid.UUID, status models.EvaluationStatus) error {
	result := r.db.Model(&models.Evaluation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("evaluation not found")
	}

	return nil
}

func (r *evaluationRepository) UpdateResult(id uuid.UUID, data *EvaluationUpdateData) error {
	updates := map[string]interface{}{
		"status":     models.StatusCompleted,
		"updated_at": time.Now(),
	}

	if data.CVMatchRate != nil {
		updates["cv_match_rate"] = *data.CVMatchRate
	}
	if data.CVFeedback != nil {
		updates["cv_feedback"] = *data.CVFeedback
	}
	if data.ProjectScore != nil {
		updates["project_score"] = *data.ProjectScore
	}
	if data.ProjectFeedback != nil {
		updates["project_feedback"] = *data.ProjectFeedback
	}
	if data.OverallSummary != nil {
		updates["overall_summary"] = *data.OverallSummary
	}

	result := r.db.Model(&models.Evaluation{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update result: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("evaluation not found")
	}

	return nil
}

func (r *evaluationRepository) UpdateError(id uuid.UUID, errorMsg string) error {
	result := r.db.Model(&models.Evaluation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        models.StatusFailed,
			"error_message": errorMsg,
			"updated_at":    time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("evaluation not found")
	}

	return nil
}

func (r *evaluationRepository) FindPendingJobs(limit int) ([]models.Evaluation, error) {
	var evals []models.Evaluation
	err := r.db.
		Where("status = ?", models.StatusQueued).
		Order("created_at ASC").
		Limit(limit).
		Find(&evals).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find pending jobs: %w", err)
	}

	return evals, nil
}
