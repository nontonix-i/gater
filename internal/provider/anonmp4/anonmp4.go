package anonmp4

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/user/gater/internal/provider"
)

type AnonMP4 struct {
	client *http.Client
}

type uploadResp struct {
	Success   bool   `json:"success"`
	VideoID   string `json:"video_id"`
	WatchURL  string `json:"watch_url"`
	EmbedURL  string `json:"embed_url"`
	DeleteURL string `json:"delete_url"`
	Title     string `json:"title"`
	Message   string `json:"message"`
}

const apiBase = "https://anonmp4api.xyz"

func New() *AnonMP4 {
	return &AnonMP4{
		client: &http.Client{Timeout: 30 * time.Minute},
	}
}

func (p *AnonMP4) Name() string             { return "anonmp4" }
func (p *AnonMP4) Type() provider.Type       { return provider.TypeVideoHost }
func (p *AnonMP4) SupportsAnonymous() bool   { return true }
func (p *AnonMP4) SupportsRemoteURL() bool   { return false }
func (p *AnonMP4) HasAPI() bool              { return true }

func (p *AnonMP4) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, file); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/upload", body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	var up uploadResp
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&up); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if !up.Success {
		return nil, fmt.Errorf("upload failed: %s", up.Message)
	}

	return &provider.Result{
		OutputURL: up.WatchURL,
		FileCode:  up.VideoID,
		FileName:  up.Title,
	}, nil
}

func (p *AnonMP4) UploadFromURL(ctx context.Context, url string, opts map[string]string) (*provider.Result, error) {
	return nil, fmt.Errorf("anonmp4 does not support remote URL upload")
}
