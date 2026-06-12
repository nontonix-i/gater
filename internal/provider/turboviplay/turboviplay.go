package turboviplay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/user/gater/internal/provider"
)

type Turboviplay struct {
	client *http.Client
}

type serverResp struct {
	Result string `json:"result"`
}

type uploadResp struct {
	VideoID json.RawMessage `json:"videoID"`
	Title   string          `json:"title"`
}

type remoteResp struct {
	VideoID string `json:"videoID"`
}

const apiBase = "https://api.turboviplay.com"

func New() *Turboviplay {
	return &Turboviplay{
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *Turboviplay) Name() string             { return "turboviplay" }
func (p *Turboviplay) Type() provider.Type       { return provider.TypeVideoHost }
func (p *Turboviplay) SupportsAnonymous() bool   { return false }
func (p *Turboviplay) SupportsRemoteURL() bool   { return true }
func (p *Turboviplay) HasAPI() bool              { return true }

func (p *Turboviplay) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	apiKey, err := p.getKey(opts)
	if err != nil {
		return nil, err
	}

	server, err := p.getUploadServer(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("keyapi", apiKey)

	if folderID, ok := opts["folder_id"]; ok {
		w.WriteField("folder_id", folderID)
	}

	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, file); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", server, body)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
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

	fileCode, err := extractFileCode(up.VideoID)
	if err != nil {
		return nil, fmt.Errorf("extract file code: %w", err)
	}

	return &provider.Result{
		OutputURL: fmt.Sprintf("https://turbovidhls.com/t/%s", fileCode),
		FileCode:  fileCode,
	}, nil
}

func (p *Turboviplay) UploadFromURL(ctx context.Context, urlStr string, opts map[string]string) (*provider.Result, error) {
	apiKey, err := p.getKey(opts)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/uploadUrl?keyApi=%s&url=%s", apiBase, apiKey, url.QueryEscape(urlStr))
	if title, ok := opts["title"]; ok {
		u += "&newTitle=" + url.QueryEscape(title)
	}
	if folder, ok := opts["folder"]; ok {
		u += "&nameFolder=" + url.QueryEscape(folder)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("create remote request: %w", err)
	}

	var r remoteResp
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote upload: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return &provider.Result{
		OutputURL: fmt.Sprintf("https://turbovidhls.com/t/%s", r.VideoID),
		FileCode:  r.VideoID,
	}, nil
}

func (p *Turboviplay) getKey(opts map[string]string) (string, error) {
	key := opts["api_key"]
	if key == "" {
		return "", fmt.Errorf("turboviplay requires api_key in options")
	}
	return key, nil
}

func extractFileCode(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var obj struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Slug, nil
	}
	return "", fmt.Errorf("unexpected videoID format: %s", string(raw))
}

func (p *Turboviplay) getUploadServer(ctx context.Context, apiKey string) (string, error) {
	u := fmt.Sprintf("%s/uploadserver?keyApi=%s", apiBase, apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}

	var s serverResp
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", err
	}

	return s.Result, nil
}
