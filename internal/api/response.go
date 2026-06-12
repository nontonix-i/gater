package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type taskResponse struct {
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	Title       string            `json:"title,omitempty"`
	FileName    string            `json:"file_name"`
	FileSize    int64             `json:"file_size"`
	SourceType  string            `json:"source_type"`
	SourceURL   string            `json:"source_url"`
	CreatedAt   interface{}       `json:"created_at"`
	UpdatedAt   interface{}       `json:"updated_at"`
	CompletedAt *time.Time        `json:"completed_at"`
	Results     []resultResponse  `json:"results"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type resultResponse struct {
	Provider         string     `json:"provider"`
	Status           string     `json:"status"`
	Progress         int        `json:"progress"`
	SourceURL        string     `json:"source_url,omitempty"`
	OutputURL        string     `json:"output_url,omitempty"`
	FileCode         string     `json:"file_code,omitempty"`
	ProviderFileName string     `json:"provider_file_name,omitempty"`
	ProviderFileSize int64      `json:"provider_file_size,omitempty"`
	Error            string     `json:"error,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
