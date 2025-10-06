package config

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"alfredoptarigan/cv-evaluator/internal/models"
)

func InitDatabase(cfg *Config) (*gorm.DB, error) {
	dsn := cfg.GetDatabaseDSN()

	logLevel := logger.Silent
	if cfg.Server.Env == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("✅ Database connected successfully")

	// Auto migrate
	if err := db.AutoMigrate(
		&models.Document{},
		&models.Evaluation{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("✅ Database migration completed")

	return db, nil
}
