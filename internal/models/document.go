package models

import (
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Filename         string    `gorm:"type:text" json:"filename"`
	OriginalFileName string    `gorm:"type:text" json:"original_filename"`
	FileType         string    `gorm:"type:text" json:"file_type"`
	FilePath         string    `gorm:"type:text" json:"file_path"`
	CreatedAt        time.Time `gorm:"type:timestamp;default:now()" json:"created_at"`
	UpdatedAt        time.Time `gorm:"type:timestamp;default:now()" json:"updated_at"`
}

func (d *Document) TableName() string {
	return "documents"
}
