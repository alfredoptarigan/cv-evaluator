-- +goose Up
-- +goose StatementBegin
CREATE TABLE evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_title VARCHAR(255) NOT NULL,
    cv_document_id UUID REFERENCES documents(id),
    project_document_id UUID REFERENCES documents(id),
    status VARCHAR(50) NOT NULL, -- 'queued', 'processing', 'completed', 'failed'
    cv_match_rate DECIMAL(3,2),
    cv_feedback TEXT,
    project_score DECIMAL(3,2),
    project_feedback TEXT,
    overall_summary TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_evaluations_status ON evaluations(status);
CREATE INDEX idx_evaluations_created_at ON evaluations(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS evaluations;
-- +goose StatementEnd
