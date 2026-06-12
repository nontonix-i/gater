package lulustream

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

type LuluStream struct {
	client *http.Client
}

type serverResp struct {
	Result string `json:"result"`
}

type uploadResp struct {
	Files []struct {
		FileCode string `json:"filecode"`
		FileName string `json:"filename"`
		Status   string `json:"status"`
	} `json:"files"`
}

type remoteResp struct {
	Result struct {
		FileCode string `json:"filecode"`
	} `json:"result"`
}

const apiBase = "https://api.lulustream.com/api"

func New() *LuluStream {
	return &LuluStream{
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *LuluStream) Name() string             { return "lulustream" }
func (p *LuluStream) Type() provider.Type       { return provider.TypeVideoHost }
func (p *LuluStream) SupportsAnonymous() bool   { return false }
func (p *LuluStream) SupportsRemoteURL() bool   { return true }
func (p *LuluStream) HasAPI() bool              { return true }

func (p *LuluStream) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
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
	w.WriteField("key", apiKey)

	if title, ok := opts["title"]; ok {
		w.WriteField("file_title", title)
	}
	if folderID, ok := opts["folder_id"]; ok {
		w.WriteField("fld_id", folderID)
	}
	if catID, ok := opts["category_id"]; ok {
		w.WriteField("cat_id", catID)
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

	if len(up.Files) == 0 {
		return nil, fmt.Errorf("no files in upload response")
	}

	f := up.Files[0]
	if f.Status != "OK" {
		return nil, fmt.Errorf("upload failed: status=%s", f.Status)
	}

	return &provider.Result{
		OutputURL: fmt.Sprintf("https://lulustream.com/%s.html", f.FileCode),
		FileCode:  f.FileCode,
		FileName:  f.FileName,
	}, nil
}

func (p *LuluStream) UploadFromURL(ctx context.Context, urlStr string, opts map[string]string) (*provider.Result, error) {
	apiKey, err := p.getKey(opts)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/upload/url?key=%s&url=%s", apiBase, apiKey, url.QueryEscape(urlStr))
	if folderID, ok := opts["folder_id"]; ok {
		u += "&fld_id=" + url.QueryEscape(folderID)
	}
	if catID, ok := opts["category_id"]; ok {
		u += "&cat_id=" + url.QueryEscape(catID)
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

	fileCode := r.Result.FileCode
	if fileCode == "" {
		return nil, fmt.Errorf("no filecode in response")
	}

	return &provider.Result{
		OutputURL: fmt.Sprintf("https://lulustream.com/%s.html", fileCode),
		FileCode:  fileCode,
	}, nil
}

func (p *LuluStream) getKey(opts map[string]string) (string, error) {
	key := opts["api_key"]
	if key == "" {
		return "", fmt.Errorf("lulustream requires api_key in options")
	}
	return key, nil
}

func (p *LuluStream) getUploadServer(ctx context.Context, apiKey string) (string, error) {
	u := fmt.Sprintf("%s/upload/server?key=%s", apiBase, apiKey)
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

	if s.Result == "" {
		return "", fmt.Errorf("empty server URL")
	}

	return s.Result, nil
}
