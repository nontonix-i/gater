package database

import (
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/user/gater/internal/model"
)

func New(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	slog.Info("running auto-migration")
	if err := db.AutoMigrate(
		&model.User{},
		&model.ProviderCredential{},
		&model.Task{},
		&model.TaskResult{},
		&model.TaskMetadata{},
	); err != nil {
		return nil, err
	}

	return db, nil
}
