package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"alfredoptarigan/cv-evaluator/internal/models"
	"alfredoptarigan/cv-evaluator/internal/repositories"
	"alfredoptarigan/cv-evaluator/internal/services"
)

type UploadHandler struct {
	docRepo        repositories.DocumentRepository
	storageService services.StorageService
	maxFileSize    int64
}

func NewUploadHandler(
	docRepo repositories.DocumentRepository,
	storageService services.StorageService,
	maxFileSize int64,
) *UploadHandler {
	return &UploadHandler{
		docRepo:        docRepo,
		storageService: storageService,
		maxFileSize:    maxFileSize,
	}
}

func (h *UploadHandler) HandleUpload(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to parse multipart form",
		})
	}

	files := form.File

	var responses []models.UploadResponse

	// Process the cv file
	if cvFiles, exists := files["cv"]; exists && len(cvFiles) > 0 {
		cvFile := cvFiles[0]

		if cvFile.Size > h.maxFileSize {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("CV file too large. Max size: %d bytes", h.maxFileSize),
			})
		}

		// Save file
		filename, filePath, err := h.storageService.SaveFile(cvFile, "cv")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to save CV file: %v", err),
			})
		}

		// Create document record
		doc := models.Document{
			ID:               uuid.New(),
			Filename:         filename,
			OriginalFileName: cvFile.Filename,
			FileType:         "cv",
			FilePath:         filePath,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		// Save document to repository
		if err := h.docRepo.Create(&doc); err != nil {
			// Cleanup uploaded file if database insert fails
			h.storageService.DeleteFile(filename)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to save CV document record: %v", err),
			})
		}

		responses = append(responses, models.UploadResponse{
			ID:           doc.ID.String(),
			Filename:     doc.Filename,
			OriginalName: doc.OriginalFileName,
			FileType:     doc.FileType,
		})
	}

	// Process the project report
	if projectFiles, exists := files["project_report"]; exists && len(projectFiles) > 0 {
		projectFile := projectFiles[0]

		if projectFile.Size > h.maxFileSize {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Project report file too large. Max size: %d bytes", h.maxFileSize),
			})
		}

		// Save file
		filename, filePath, err := h.storageService.SaveFile(projectFile, "project_report")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to save project report file: %v", err),
			})
		}

		// Save document record
		doc := models.Document{
			ID:               uuid.New(),
			Filename:         filename,
			OriginalFileName: projectFile.Filename,
			FileType:         "project_report",
			FilePath:         filePath,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := h.docRepo.Create(&doc); err != nil {
			// Cleanup uploaded file if database insert fails
			h.storageService.DeleteFile(filename)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save project report document record",
			})
		}

		responses = append(responses, models.UploadResponse{
			ID:           doc.ID.String(),
			Filename:     doc.Filename,
			OriginalName: doc.OriginalFileName,
			FileType:     doc.FileType,
		})
	}

	if len(responses) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No valid files uploaded. Please upload 'cv' and/or 'project_report' as PDF files.",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":   "Files uploaded successfully",
		"documents": responses,
	})
}
