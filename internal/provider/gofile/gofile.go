package gofile

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

type GoFile struct {
	client *http.Client
}

type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

type uploadResult struct {
	DownloadPage     string `json:"downloadPage"`
	ID               string `json:"id"`
	ParentFolder     string `json:"parentFolder"`
	ParentFolderCode string `json:"parentFolderCode"`
	Name             string `json:"name"`
	Size             int64  `json:"size"`
}

const uploadURL = "https://upload.gofile.io/uploadfile"

func New() *GoFile {
	return &GoFile{
		client: &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *GoFile) Name() string             { return "gofile" }
func (p *GoFile) Type() provider.Type       { return provider.TypeStorage }
func (p *GoFile) SupportsAnonymous() bool   { return true }
func (p *GoFile) SupportsRemoteURL() bool   { return false }
func (p *GoFile) HasAPI() bool              { return true }

func (p *GoFile) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(fw, file); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}

	if folderID, ok := opts["folder_id"]; ok {
		w.WriteField("folderId", folderID)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	if token, ok := opts["token"]; ok {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Status != "ok" {
		return nil, fmt.Errorf("api error: status=%s", apiResp.Status)
	}

	var result uploadResult
	if err := json.Unmarshal(apiResp.Data, &result); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}

	return &provider.Result{
		OutputURL: result.DownloadPage,
		FileCode:  result.ID,
		FileName:  result.Name,
		FileSize:  result.Size,
	}, nil
}

func (p *GoFile) UploadFromURL(ctx context.Context, url string, opts map[string]string) (*provider.Result, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	filename := "remote_file"
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, err := fmt.Sscanf(cd, "filename=\"%s\"", &filename); err != nil {
			filename = "remote_file"
		}
	}

	return p.Upload(ctx, resp.Body, filename, opts)
}
