package seekstreaming

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/user/gater/internal/provider"
)

type SeekStreaming struct {
	client *http.Client
}

type uploadEndpointResp struct {
	TusURL      string `json:"tusUrl"`
	AccessToken string `json:"accessToken"`
}

type advanceUploadResp struct {
	ID string `json:"id"`
}

type advanceTaskResp struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Videos []string `json:"videos"`
	Error  string   `json:"error,omitempty"`
}

const apiBase = "https://seekstreaming.com/api/v1"

func New() *SeekStreaming {
	return &SeekStreaming{
		client: &http.Client{Timeout: 30 * time.Minute},
	}
}

func (p *SeekStreaming) Name() string             { return "seekstreaming" }
func (p *SeekStreaming) Type() provider.Type       { return provider.TypeVideoHost }
func (p *SeekStreaming) SupportsAnonymous() bool   { return false }
func (p *SeekStreaming) SupportsRemoteURL() bool   { return true }
func (p *SeekStreaming) HasAPI() bool              { return true }

func (p *SeekStreaming) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	token, err := p.getToken(opts)
	if err != nil {
		return nil, err
	}

	ep, err := p.getUploadEndpoint(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get endpoint: %w", err)
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	fileSize := int64(len(fileData))

	uploadURL, err := p.createTUSUpload(ctx, ep.TusURL, ep.AccessToken, filename, fileSize)
	if err != nil {
		return nil, fmt.Errorf("create tus upload: %w", err)
	}

	videoID, err := p.uploadTUSData(ctx, uploadURL, ep.AccessToken, fileData)
	if err != nil {
		return nil, fmt.Errorf("upload tus data: %w", err)
	}

	return &provider.Result{
		OutputURL: fmt.Sprintf("https://seekstreaming.com/v/%s", videoID),
		FileCode:  videoID,
		FileName:  filename,
		FileSize:  fileSize,
	}, nil
}

func (p *SeekStreaming) UploadFromURL(ctx context.Context, urlStr string, opts map[string]string) (*provider.Result, error) {
	token, err := p.getToken(opts)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"url": urlStr,
	}
	if name, ok := opts["name"]; ok {
		body["name"] = name
	}
	if folderID, ok := opts["folder_id"]; ok {
		body["folderId"] = folderID
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/video/advance-upload", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("create advance upload: %w", err)
	}
	req.Header.Set("api-token", token)
	req.Header.Set("Content-Type", "application/json")

	var adv advanceUploadResp
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("advance upload: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&adv); err != nil {
		return nil, fmt.Errorf("decode advance upload: %w", err)
	}

	taskID := adv.ID

	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}

		if fn, ok := provider.GetProgress(ctx); ok {
			fn(5+i*3, "processing")
		}

		task, err := p.getAdvanceTask(ctx, token, taskID)
		if err != nil {
			return nil, fmt.Errorf("check task: %w", err)
		}

		switch task.Status {
		case "Completed":
			if len(task.Videos) > 0 {
				videoID := task.Videos[0]
				if fn, ok := provider.GetProgress(ctx); ok {
					fn(100, "completed")
				}
				return &provider.Result{
					OutputURL: fmt.Sprintf("https://seekstreaming.com/v/%s", videoID),
					FileCode:  videoID,
				}, nil
			}
			return nil, fmt.Errorf("no videos in completed task")
		case "Failed", "Error":
			errMsg := task.Error
			if errMsg == "" {
				errMsg = "advance upload failed"
			}
			return nil, fmt.Errorf("%s", errMsg)
		}
	}

	return nil, fmt.Errorf("advance upload timed out")
}

func (p *SeekStreaming) getToken(opts map[string]string) (string, error) {
	token := opts["api_token"]
	if token == "" {
		return "", fmt.Errorf("seekstreaming requires api_token in options")
	}
	return token, nil
}

func (p *SeekStreaming) getUploadEndpoint(ctx context.Context, token string) (*uploadEndpointResp, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+"/video/upload", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-token", token)

	var ep uploadEndpointResp
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&ep); err != nil {
		return nil, err
	}

	return &ep, nil
}

func (p *SeekStreaming) createTUSUpload(ctx context.Context, tusURL, accessToken, filename string, fileSize int64) (string, error) {
	b64Name := base64.StdEncoding.EncodeToString([]byte(filename))
	b64Access := base64.StdEncoding.EncodeToString([]byte(accessToken))
	b64Type := base64.StdEncoding.EncodeToString([]byte("mp4"))

	req, err := http.NewRequestWithContext(ctx, "POST", tusURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set("Upload-Length", fmt.Sprintf("%d", fileSize))
	req.Header.Set("Upload-Metadata", fmt.Sprintf("filename %s,accessToken %s,filetype %s", b64Name, b64Access, b64Type))

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create tus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create tus failed: %d %s", resp.StatusCode, string(body))
	}

	uploadURL := resp.Header.Get("Location")
	if uploadURL == "" {
		return "", fmt.Errorf("no Location header in TUS create response")
	}

	return uploadURL, nil
}

func (p *SeekStreaming) uploadTUSData(ctx context.Context, uploadURL, accessToken string, fileData []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "PATCH", uploadURL, bytes.NewReader(fileData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set("Content-Type", "application/offset+octet-stream")
	req.Header.Set("Upload-Offset", "0")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tus patch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tus upload failed: %d %s", resp.StatusCode, string(body))
	}

	tusURL := resp.Request.URL.String()
	videoID := extractVideoID(tusURL)

	if videoID == "" {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("could not extract video ID from TUS response: %s", string(body))
	}

	return videoID, nil
}

func (p *SeekStreaming) getAdvanceTask(ctx context.Context, token, taskID string) (*advanceTaskResp, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+"/video/advance-upload/"+taskID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-token", token)

	var task advanceTaskResp
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}

	return &task, nil
}

func extractVideoID(urlStr string) string {
	for i := len(urlStr) - 1; i >= 0; i-- {
		if urlStr[i] == '/' {
			return urlStr[i+1:]
		}
	}
	return urlStr
}
