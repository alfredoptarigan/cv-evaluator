package models

type UploadResponse struct {
	ID           string `json:"id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	FileType     string `json:"file_type"`
}

type EvaluateRequest struct {
	JobTitle          string `json:"job_title" validate:"required"`
	CVDocumentID      string `json:"cv_document_id" validate:"required,uuid"`
	ProjectDocumentID string `json:"project_document_id" validate:"required,uuid"`
}

type EvaluateResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ResultResponse struct {
	ID           string          `json:"id"`
	Status       string          `json:"status"`
	Result       *EvaluationData `json:"result,omitempty"`
	ErrorMessage *string         `json:"error_message,omitempty"`
}

type EvaluationData struct {
	CVMatchRate     float64 `json:"cv_match_rate"`
	CVFeedback      string  `json:"cv_feedback"`
	ProjectScore    float64 `json:"project_score"`
	ProjectFeedback string  `json:"project_feedback"`
	OverallSummary  string  `json:"overall_summary"`
}
