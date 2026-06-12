package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/user/gater/internal/model"
	"github.com/user/gater/internal/provider"
)

type CreateRequest struct {
	UserID    string
	SourceType string
	SourceURL string
	Title     string
	FileName  string
	FileSize  int64
	File      io.Reader
	Providers []string
	Metadata  map[string]string
}

type Manager struct {
	db       *gorm.DB
	registry *provider.Registry
	tempDir  string
}

func NewManager(db *gorm.DB, reg *provider.Registry, tempDir string) *Manager {
	return &Manager{
		db:       db,
		registry: reg,
		tempDir:  tempDir,
	}
}

func (m *Manager) Create(ctx context.Context, req *CreateRequest) (*model.Task, error) {
	var filePath string

	if req.File != nil {
		if err := os.MkdirAll(m.tempDir, 0755); err != nil {
			return nil, fmt.Errorf("create temp dir: %w", err)
		}

		ext := filepath.Ext(req.FileName)
		filePath = filepath.Join(m.tempDir, uuid.New().String()+ext)

		f, err := os.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("create temp file: %w", err)
		}

		written, err := io.Copy(f, req.File)
		if err != nil {
			f.Close()
			os.Remove(filePath)
			return nil, fmt.Errorf("save temp file: %w", err)
		}
		f.Close()

		if req.FileSize == 0 {
			req.FileSize = written
		}
	}

	task := &model.Task{
		UserID:     req.UserID,
		Status:     "pending",
		SourceType: req.SourceType,
		SourceURL:  req.SourceURL,
		Title:      req.Title,
		FileName:   req.FileName,
		FileSize:   req.FileSize,
		FilePath:   filePath,
	}

	if err := m.db.Create(task).Error; err != nil {
		if filePath != "" {
			os.Remove(filePath)
		}
		return nil, fmt.Errorf("create task: %w", err)
	}

	for _, p := range req.Providers {
		result := &model.TaskResult{
			TaskID:   task.ID,
			Provider: p,
			Status:   "pending",
		}
		if err := m.db.Create(result).Error; err != nil {
			return nil, fmt.Errorf("create task result: %w", err)
		}
	}

	for k, v := range req.Metadata {
		meta := &model.TaskMetadata{
			TaskID: task.ID,
			Key:    k,
			Value:  v,
		}
		if err := m.db.Create(meta).Error; err != nil {
			return nil, fmt.Errorf("create metadata: %w", err)
		}
	}

	go m.Process(task.ID, req.Providers)

	return task, nil
}

func (m *Manager) GetTask(taskID string) (*model.Task, error) {
	var task model.Task
	err := m.db.Preload("Results").Preload("Metadata").First(&task, "id = ?", taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (m *Manager) ListTasks(userID string, limit, offset int) ([]model.Task, int64, error) {
	var tasks []model.Task
	var total int64

	query := m.db.Model(&model.Task{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	query.Count(&total)

	if err := query.Preload("Results").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}
