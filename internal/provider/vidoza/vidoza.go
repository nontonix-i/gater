package vidoza

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/user/gater/internal/provider"
)

type Vidoza struct {
	client *http.Client
}

type serverResp struct {
	Data struct {
		UploadURL    string            `json:"upload_url"`
		UploadParams map[string]string `json:"upload_params"`
	} `json:"data"`
}

type uploadResult struct {
	Status string `json:"status"`
	Code   string `json:"code"`
}

type remoteResp struct {
	ID       int    `json:"id"`
	URL      string `json:"url"`
	Status   string `json:"status"`
	FileCode string `json:"file_code"`
}

const apiBase = "https://api.vidoza.net"

func New() *Vidoza {
	return &Vidoza{
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *Vidoza) Name() string             { return "vidoza" }
func (p *Vidoza) Type() provider.Type       { return provider.TypeVideoHost }
func (p *Vidoza) SupportsAnonymous() bool   { return false }
func (p *Vidoza) SupportsRemoteURL() bool   { return true }
func (p *Vidoza) HasAPI() bool              { return true }

func (p *Vidoza) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	token, err := p.getToken(opts)
	if err != nil {
		return nil, err
	}

	server, params, err := p.getUploadServer(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	for k, v := range params {
		w.WriteField(k, v)
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

	var up uploadResult
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&up); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if up.Status != "OK" {
		return nil, fmt.Errorf("upload failed: status=%s", up.Status)
	}

	return &provider.Result{
		FileCode:  up.Code,
		OutputURL: fmt.Sprintf("https://vidoza.net/%s.html", up.Code),
	}, nil
}

func (p *Vidoza) UploadFromURL(ctx context.Context, urlStr string, opts map[string]string) (*provider.Result, error) {
	token, err := p.getToken(opts)
	if err != nil {
		return nil, err
	}

	catID := opts["category_id"]
	if catID == "" {
		catID = "3" // Not adult
	}
	fldID := opts["folder_id"]
	if fldID == "" {
		fldID = "0" // root folder
	}

	form := url.Values{}
	form.Set("cat_id", catID)
	form.Set("fld_id", fldID)
	form.Set("url", urlStr)

	req, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/v1/upload/url", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create remote request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var r remoteResp
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote upload: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if r.ID == 0 {
		return nil, fmt.Errorf("remote upload failed: id=%d url=%s", r.ID, r.URL)
	}

	return &provider.Result{
		OutputURL: r.URL,
	}, nil
}

func (p *Vidoza) getToken(opts map[string]string) (string, error) {
	token := opts["api_token"]
	if token == "" {
		return "", fmt.Errorf("vidoza requires api_token in options")
	}
	return token, nil
}

func (p *Vidoza) getUploadServer(ctx context.Context, token string) (string, map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+"/v1/upload/http/server", nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var s serverResp
	resp, err := p.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", nil, err
	}

	return s.Data.UploadURL, s.Data.UploadParams, nil
}
