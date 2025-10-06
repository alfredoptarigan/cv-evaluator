package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"alfredoptarigan/cv-evaluator/internal/models"
)

type DocumentRepository interface {
	Create(document *models.Document) error
	FindByID(id uuid.UUID) (*models.Document, error)
	FindByIDs(ids []uuid.UUID) ([]models.Document, error)
}

type documentRepository struct {
	db *gorm.DB
}

// Create implements DocumentRepository.
func (d *documentRepository) Create(document *models.Document) error {
	if err := d.db.Create(&document).Error; err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	return nil
}

// FindByID implements DocumentRepository.
func (d *documentRepository) FindByID(id uuid.UUID) (*models.Document, error) {
	var doc models.Document
	if err := d.db.Where("id = ?", id).First(&doc).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("document not found: %w", err)
		}

		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	return &doc, nil
}

// FindByIDs implements DocumentRepository.
func (d *documentRepository) FindByIDs(ids []uuid.UUID) ([]models.Document, error) {
	var docs []models.Document
	if err := d.db.Where("id IN ?", ids).Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}

	return docs, nil
}

func NewDocumentRepository(db *gorm.DB) DocumentRepository {
	return &documentRepository{db: db}
}
