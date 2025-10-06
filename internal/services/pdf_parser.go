package services

import (
	"fmt"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

type PDFParserService interface {
	ExtractText(filepath string) (string, error)
	ExtractTextWithMetaData(filepath string) (*PDFContent, error)
}

type PDFContent struct {
	Text      string
	PageCount int
	FilePath  string
}

type pdfParserService struct{}

func NewPDFParserService() PDFParserService {
	return &pdfParserService{}
}

func (p *pdfParserService) ExtractText(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var textBuilder strings.Builder
	totalPage := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		page := r.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			// Log error but continue with other pages
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	text := textBuilder.String()
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("no text content found in PDF")
	}

	return text, nil
}

func (p *pdfParserService) ExtractTextWithMetaData(filePath string) (*PDFContent, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var textBuilder strings.Builder
	totalPage := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		page := r.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			// Log error but continue with other pages
			continue
		}

		textBuilder.WriteString(fmt.Sprintf("--- Page %d ---\n", pageIndex))
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	text := textBuilder.String()
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("no text content found in PDF")
	}

	return &PDFContent{
		Text:      text,
		PageCount: totalPage,
		FilePath:  filePath,
	}, nil
}

// Helper function to clean and normalize text
func CleanText(text string) string {
	// Remove excessive whitespace
	text = strings.TrimSpace(text)

	// Replace multiple newlines with double newline
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, "\n")
}
