package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/user/gater/internal/model"
	"github.com/user/gater/internal/provider"
)

func (m *Manager) Process(taskID string, providers []string) {
	slog.Info("processing task", "task_id", taskID)

	m.db.Model(&model.Task{}).Where("id = ?", taskID).Update("status", "processing")

	var wg sync.WaitGroup
	for _, name := range providers {
		wg.Add(1)
		go func(providerName string) {
			defer wg.Done()
			m.uploadToProvider(taskID, providerName)
		}(name)
	}

	wg.Wait()
	m.finalizeTask(taskID)
}

func (m *Manager) uploadToProvider(taskID, providerName string) {
	prov, err := m.registry.Get(providerName)
	if err != nil {
		m.updateResult(taskID, providerName, "failed", err.Error())
		return
	}

	if !prov.HasAPI() {
		m.updateResult(taskID, providerName, "failed", "no public API available")
		return
	}

	m.updateResultStatus(taskID, providerName, "uploading")

	var task model.Task
	if err := m.db.First(&task, "id = ?", taskID).Error; err != nil {
		m.updateResult(taskID, providerName, "failed", "task not found")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	progress := func(pct int, msg string) {
		m.db.Model(&model.TaskResult{}).
			Where("task_id = ? AND provider = ?", taskID, providerName).
			Update("progress", pct)
	}

	ctx = provider.WithProgress(ctx, progress)
	opts := m.getProviderOpts(task.UserID, providerName)
	remoteOK := prov.SupportsRemoteURL()
	isRemote := task.SourceType == "remote_url"

	var result *provider.Result

	progress(5, "starting")

	if isRemote && remoteOK {
		progress(10, "remote upload")
		result, err = prov.UploadFromURL(ctx, task.SourceURL, opts)
		if err == nil {
			progress(100, "completed")
			m.saveResult(taskID, providerName, result)
			return
		}
		slog.Warn("remote upload failed, falling back to direct upload",
			"provider", providerName, "task_id", taskID, "error", err)
	}

	filePath := task.FilePath
	if filePath == "" && task.SourceURL != "" {
		progress(10, "downloading source")
		filePath, err = m.downloadFile(ctx, task.SourceURL)
		if err != nil {
			m.updateResult(taskID, providerName, "failed",
				fmt.Sprintf("download failed: %v", err))
			return
		}
		defer os.Remove(filePath)
		progress(30, "downloaded")
	}

	if filePath == "" {
		m.updateResult(taskID, providerName, "failed", "no file available")
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		m.updateResult(taskID, providerName, "failed", fmt.Sprintf("open file: %v", err))
		return
	}
	defer file.Close()

	filename := task.FileName
	if filename == "" {
		filename = filepath.Base(filePath)
	}

	info, _ := file.Stat()
	var reader io.Reader = file
	if info != nil && info.Size() > 0 {
		reader = provider.NewProgressReader(file, info.Size(), func(pct int, msg string) {
			progress(pct, msg)
		})
	}

	result, err = prov.Upload(ctx, reader, filename, opts)
	if err != nil {
		m.updateResult(taskID, providerName, "failed", err.Error())
		return
	}

	progress(100, "completed")
	m.saveResult(taskID, providerName, result)
}

func (m *Manager) downloadFile(ctx context.Context, sourceURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", sourceURL, nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}

	if err := os.MkdirAll(m.tempDir, 0755); err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	ext := ".tmp"
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, err := fmt.Sscanf(cd, `filename="%s"`, &ext); err == nil {
			ext = filepath.Ext(ext)
		}
	}

	filePath := filepath.Join(m.tempDir, uuid.New().String()+ext)
	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("save download: %w", err)
	}

	return filePath, nil
}

func (m *Manager) saveResult(taskID, providerName string, result *provider.Result) {
	update := map[string]interface{}{
		"status":              "completed",
		"output_url":          result.OutputURL,
		"file_code":           result.FileCode,
		"provider_file_name":  result.FileName,
		"provider_file_size":  result.FileSize,
		"completed_at":        time.Now(),
	}
	m.db.Model(&model.TaskResult{}).
		Where("task_id = ? AND provider = ?", taskID, providerName).
		Updates(update)

	slog.Info("upload completed", "task_id", taskID, "provider", providerName)
}

func (m *Manager) finalizeTask(taskID string) {
	var results []model.TaskResult
	m.db.Where("task_id = ?", taskID).Find(&results)

	total := len(results)
	completed := 0
	failed := 0

	for _, r := range results {
		switch r.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	status := "completed"
	if failed == total {
		status = "failed"
	} else if failed > 0 {
		status = "partial"
	}

	now := time.Now()
	m.db.Model(&model.Task{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":       status,
		"completed_at": now,
		"updated_at":   now,
	})

	if task, _ := m.getTask(taskID); task != nil && task.FilePath != "" {
		os.Remove(task.FilePath)
	}

	slog.Info("task completed", "task_id", taskID, "status", status,
		"completed", completed, "failed", failed)
}

func (m *Manager) updateResult(taskID, providerName, status, errMsg string) {
	now := time.Now()
	m.db.Model(&model.TaskResult{}).
		Where("task_id = ? AND provider = ?", taskID, providerName).
		Updates(map[string]interface{}{
			"status":        status,
			"error_message": errMsg,
			"completed_at":  now,
		})
}

func (m *Manager) updateResultStatus(taskID, providerName, status string) {
	now := time.Now()
	m.db.Model(&model.TaskResult{}).
		Where("task_id = ? AND provider = ?", taskID, providerName).
		Updates(map[string]interface{}{
			"status":     status,
			"started_at": now,
		})
}

func (m *Manager) getProviderOpts(userID, providerName string) map[string]string {
	opts := make(map[string]string)

	var cred model.ProviderCredential
	err := m.db.Where("user_id = ? AND provider = ?", userID, providerName).First(&cred).Error
	if err != nil {
		err = m.db.Where("provider = ?", providerName).First(&cred).Error
	}
	if err == nil {
		data, _ := cred.Credentials.MarshalJSON()
		var parsed map[string]string
		if err := json.Unmarshal(data, &parsed); err == nil {
			for k, v := range parsed {
				opts[k] = v
			}
		}
	}

	return opts
}

func (m *Manager) getTask(taskID string) (*model.Task, error) {
	var task model.Task
	err := m.db.Preload("Results").Preload("Metadata").First(&task, "id = ?", taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}
