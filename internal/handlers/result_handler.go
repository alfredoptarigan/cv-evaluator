package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"alfredoptarigan/cv-evaluator/internal/models"
	"alfredoptarigan/cv-evaluator/internal/repositories"
)

type ResultHandler struct {
	evalRepo repositories.EvaluationRepository
}

func NewResultHandler(evalRepo repositories.EvaluationRepository) *ResultHandler {
	return &ResultHandler{
		evalRepo: evalRepo,
	}
}

func (h *ResultHandler) HandleGetResult(c *fiber.Ctx) error {
	// Parse ID from params
	idParam := c.Params("id")
	evalID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid evaluation ID format",
		})
	}

	// Get evaluation
	evaluation, err := h.evalRepo.FindByID(evalID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Evaluation not found",
		})
	}

	// Build response based on status
	response := models.ResultResponse{
		ID:     evaluation.ID.String(),
		Status: string(evaluation.Status),
	}

	// If completed, include results
	if evaluation.Status == models.StatusCompleted {
		response.Result = &models.EvaluationData{
			CVMatchRate:     evaluation.CVMatchRate,
			CVFeedback:      evaluation.CVFeedback,
			ProjectScore:    evaluation.ProjectScore,
			ProjectFeedback: evaluation.ProjectFeedback,
			OverallSummary:  evaluation.OverallSummary,
		}
	}

	// If failed, include error message
	if evaluation.Status == models.StatusFailed && evaluation.ErrorMessage != "" {
		response.ErrorMessage = &evaluation.ErrorMessage
	}

	return c.JSON(response)
}
