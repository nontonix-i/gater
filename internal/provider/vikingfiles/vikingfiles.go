package vikingfiles

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/user/gater/internal/provider"
)

type VikingFiles struct {
	client *http.Client
}

type serverResp struct {
	Server string `json:"server"`
}

type uploadResp struct {
	Name string `json:"name"`
	Size string `json:"size"`
	Hash string `json:"hash"`
	URL  string `json:"url"`
}

const apiBase = "https://vikingfile.com/api"

func New() *VikingFiles {
	return &VikingFiles{
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *VikingFiles) Name() string             { return "vikingfiles" }
func (p *VikingFiles) Type() provider.Type       { return provider.TypeStorage }
func (p *VikingFiles) SupportsAnonymous() bool   { return true }
func (p *VikingFiles) SupportsRemoteURL() bool   { return true }
func (p *VikingFiles) HasAPI() bool              { return true }

func (p *VikingFiles) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	server, err := p.getServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	user := opts["user"]
	w.WriteField("user", user)

	if path, ok := opts["path"]; ok {
		w.WriteField("path", path)
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

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	up, err := parseResult(ctx, respBody)
	if err != nil {
		return nil, fmt.Errorf("decode upload result: %w", err)
	}

	if up.URL == "" {
		return nil, fmt.Errorf("upload failed: %s", string(respBody))
	}

	return &provider.Result{
		OutputURL: up.URL,
		FileCode:  up.Hash,
		FileName:  up.Name,
	}, nil
}

func (p *VikingFiles) UploadFromURL(ctx context.Context, urlStr string, opts map[string]string) (*provider.Result, error) {
	server, err := p.getServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("link", urlStr)
	w.WriteField("user", opts["user"])
	if name, ok := opts["name"]; ok {
		w.WriteField("name", name)
	}
	if path, ok := opts["path"]; ok {
		w.WriteField("path", path)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", server, body)
	if err != nil {
		return nil, fmt.Errorf("create remote request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote upload: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	up, err := parseResult(ctx, respBody)
	if err != nil {
		return nil, fmt.Errorf("decode remote result: %w", err)
	}

	if up.URL == "" {
		return nil, fmt.Errorf("remote upload failed: %s", string(respBody))
	}

	return &provider.Result{
		OutputURL: up.URL,
		FileCode:  up.Hash,
		FileName:  up.Name,
	}, nil
}

type progressLine struct {
	Progress string `json:"progress"`
	Current  int    `json:"current"`
	Total    int    `json:"total"`
	Name     string `json:"name"`
}

func parseResult(ctx context.Context, body []byte) (*uploadResp, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	var last *uploadResp
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var pl progressLine
		if err := json.Unmarshal([]byte(line), &pl); err == nil && pl.Progress != "" {
			if fn, ok := provider.GetProgress(ctx); ok {
				fn(pl.Current*100/pl.Total, pl.Progress)
			}
			continue
		}

		var r uploadResp
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		if r.URL != "" {
			last = &r
		}
	}
	if last != nil {
		return last, nil
	}
	var r uploadResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (p *VikingFiles) getServer(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+"/get-server", nil)
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

	return s.Server, nil
}
