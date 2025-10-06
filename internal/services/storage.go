package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type StorageService interface {
	SaveFile(file *multipart.FileHeader, fileType string) (string, string, error)
	GetFilePath(filename string) string
	DeleteFile(filename string) error
	EnsureUploadDir() error
}

type storageService struct {
	uploadPath string
}

func NewStorageService(uploadPath string) StorageService {
	return &storageService{
		uploadPath: uploadPath,
	}
}

func (s *storageService) EnsureUploadDir() error {
	if err := os.MkdirAll(s.uploadPath, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	return nil
}

func (s *storageService) SaveFile(file *multipart.FileHeader, fileType string) (string, string, error) {
	// Validate file extensions
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" {
		return "", "", fmt.Errorf("invalid file extension: %s", ext)
	}

	// Generate the unique filename
	uniqueFilename := fmt.Sprintf("%s_%s%s", fileType, uuid.New().String(), ext)
	filePath := filepath.Join(s.uploadPath, uniqueFilename)

	// Open source file
	src, err := file.Open()
	if err != nil {
		return "", "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file
	if _, err := io.Copy(dst, src); err != nil {
		return "", "", fmt.Errorf("failed to save file: %w", err)
	}

	return uniqueFilename, filePath, nil
}

func (s *storageService) GetFilePath(filename string) string {
	return filepath.Join(s.uploadPath, filename)
}

func (s *storageService) DeleteFile(filename string) error {
	filePath := s.GetFilePath(filename)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
