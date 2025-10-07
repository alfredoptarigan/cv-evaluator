package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"alfredoptarigan/cv-evaluator/internal/models"
	"alfredoptarigan/cv-evaluator/internal/repositories"
	"alfredoptarigan/cv-evaluator/internal/services"
)

type EvaluationHandler struct {
	evalRepo repositories.EvaluationRepository
	docRepo  repositories.DocumentRepository
	worker   services.Worker
}

func NewEvaluationHandler(
	evalRepo repositories.EvaluationRepository,
	docRepo repositories.DocumentRepository,
	worker services.Worker,
) *EvaluationHandler {
	return &EvaluationHandler{
		evalRepo: evalRepo,
		docRepo:  docRepo,
		worker:   worker,
	}
}

// HandleEvaluate handles POST /evaluate
func (h *EvaluationHandler) HandleEvaluate(c *fiber.Ctx) error {
	var req models.EvaluateRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if req.JobTitle == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "job_title is required",
		})
	}

	if req.CVDocumentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cv_document_id is required",
		})
	}

	if req.ProjectDocumentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_document_id is required",
		})
	}

	// Parse UUIDs
	cvDocID, err := uuid.Parse(req.CVDocumentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid cv_document_id format",
		})
	}

	projectDocID, err := uuid.Parse(req.ProjectDocumentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid project_document_id format",
		})
	}

	// Verify documents exist
	if _, err := h.docRepo.FindByID(cvDocID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "CV document not found",
		})
	}

	if _, err := h.docRepo.FindByID(projectDocID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Project document not found",
		})
	}

	// Create evaluation record
	evaluation := &models.Evaluation{
		ID:                uuid.New(),
		JobTitle:          req.JobTitle,
		CVDocumentID:      cvDocID,
		ProjectDocumentID: projectDocID,
		Status:            models.StatusQueued,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := h.evalRepo.Create(evaluation); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create evaluation job",
		})
	}

	// Enqueue job to worker
	h.worker.EnqueueJob(evaluation.ID)

	// Return job ID immediately
	return c.Status(fiber.StatusAccepted).JSON(models.EvaluateResponse{
		ID:     evaluation.ID.String(),
		Status: string(models.StatusQueued),
	})

}
