package abyss

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

type Abyss struct {
	client *http.Client
}

type uploadResp struct {
	Slug string `json:"slug"`
}

func New() *Abyss {
	return &Abyss{
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *Abyss) Name() string             { return "abyss" }
func (p *Abyss) Type() provider.Type       { return provider.TypeVideoHost }
func (p *Abyss) SupportsAnonymous() bool   { return false }
func (p *Abyss) SupportsRemoteURL() bool   { return false }
func (p *Abyss) HasAPI() bool              { return true }

func (p *Abyss) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	apiKey, err := p.getKey(opts)
	if err != nil {
		return nil, err
	}

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

	uploadURL := fmt.Sprintf("http://up.abyss.to/%s", apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
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

	if up.Slug == "" {
		return nil, fmt.Errorf("no slug in response")
	}

	return &provider.Result{
		OutputURL: fmt.Sprintf("https://abyss.to/v/%s", up.Slug),
		FileCode:  up.Slug,
	}, nil
}

func (p *Abyss) UploadFromURL(ctx context.Context, url string, opts map[string]string) (*provider.Result, error) {
	return nil, fmt.Errorf("abyss.to does not support remote URL upload")
}

func (p *Abyss) getKey(opts map[string]string) (string, error) {
	key := opts["api_key"]
	if key == "" {
		return "", fmt.Errorf("abyss requires api_key in options")
	}
	return key, nil
}
