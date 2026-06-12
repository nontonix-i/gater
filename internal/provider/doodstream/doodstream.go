package doodstream

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

type DoodStream struct {
	client *http.Client
}

type serverResp struct {
	Result string `json:"result"`
}

type uploadResp struct {
	Result struct {
		DownloadURL string `json:"download_url"`
		FileCode    string `json:"filecode"`
		FileName    string `json:"filename"`
		Size        int64  `json:"size"`
	} `json:"result"`
}

type remoteResp struct {
	Result struct {
		FileCode string `json:"filecode"`
	} `json:"result"`
}

const baseURL = "https://doodapi.co/api"

func New() *DoodStream {
	return &DoodStream{
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *DoodStream) Name() string             { return "doodstream" }
func (p *DoodStream) Type() provider.Type       { return provider.TypeVideoHost }
func (p *DoodStream) SupportsAnonymous() bool   { return false }
func (p *DoodStream) SupportsRemoteURL() bool   { return true }
func (p *DoodStream) HasAPI() bool              { return true }

func (p *DoodStream) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	apiKey, err := p.getKey(opts)
	if err != nil {
		return nil, err
	}

	server, err := p.getServer(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("api_key", apiKey)

	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, file); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}
	w.Close()

	uploadURL := server
	if folderID, ok := opts["folder_id"]; ok {
		uploadURL += "?fld_id=" + url.QueryEscape(folderID)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	var up uploadResp
	if err := p.doJSON(req, &up); err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}

	return &provider.Result{
		OutputURL: up.Result.DownloadURL,
		FileCode:  up.Result.FileCode,
		FileName:  up.Result.FileName,
		FileSize:  up.Result.Size,
	}, nil
}

func (p *DoodStream) UploadFromURL(ctx context.Context, urlStr string, opts map[string]string) (*provider.Result, error) {
	apiKey, err := p.getKey(opts)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/upload/url?key=%s&url=%s", baseURL, apiKey, url.QueryEscape(urlStr))
	if title, ok := opts["title"]; ok {
		u += "&new_title=" + url.QueryEscape(title)
	}
	if fld, ok := opts["folder_id"]; ok {
		u += "&fld_id=" + url.QueryEscape(fld)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("create remote request: %w", err)
	}

	var r remoteResp
	if err := p.doJSON(req, &r); err != nil {
		return nil, fmt.Errorf("remote upload: %w", err)
	}

	fileCode := r.Result.FileCode
	return &provider.Result{
		OutputURL: fmt.Sprintf("https://doodstream.com/d/%s", fileCode),
		FileCode:  fileCode,
	}, nil
}

func (p *DoodStream) getKey(opts map[string]string) (string, error) {
	key := opts["api_key"]
	if key == "" {
		return "", fmt.Errorf("doodstream requires api_key in options")
	}
	return key, nil
}

func (p *DoodStream) getServer(ctx context.Context, apiKey string) (string, error) {
	u := fmt.Sprintf("%s/upload/server?key=%s", baseURL, apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}

	var s serverResp
	if err := p.doJSON(req, &s); err != nil {
		return "", err
	}

	return s.Result, nil
}

func (p *DoodStream) doJSON(req *http.Request, v interface{}) error {
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(v)
}
