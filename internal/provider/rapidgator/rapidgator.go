package rapidgator

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/user/gater/internal/provider"
)

type RapidGator struct {
	client *http.Client
}

type loginResp struct {
	Response struct {
		Token string `json:"token"`
	} `json:"response"`
}

type uploadResp struct {
	Response struct {
		UploadURL  string `json:"upload_url"`
		UploadID   string `json:"upload_id"`
		FileURL    string `json:"file_url"`
		Hash       string `json:"hash"`
	} `json:"response"`
}

type remoteResp struct {
	Response struct {
		ID int `json:"id"`
	} `json:"response"`
}

const apiBase = "https://rapidgator.net/api/v2"

func New() *RapidGator {
	return &RapidGator{
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *RapidGator) Name() string             { return "rapidgator" }
func (p *RapidGator) Type() provider.Type       { return provider.TypeStorage }
func (p *RapidGator) SupportsAnonymous() bool   { return false }
func (p *RapidGator) SupportsRemoteURL() bool   { return true }
func (p *RapidGator) HasAPI() bool              { return true }

func (p *RapidGator) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	token, err := p.getToken(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	fileSize := int64(len(fileData))
	fileHash := fmt.Sprintf("%x", md5.Sum(fileData))

	u := fmt.Sprintf("%s/file/upload?token=%s&name=%s&size=%d&hash=%s",
		apiBase, token, url.QueryEscape(filename), fileSize, fileHash)

	if folderID, ok := opts["folder_id"]; ok {
		u += "&folder_id=" + url.QueryEscape(folderID)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("init upload: %w", err)
	}

	var up uploadResp
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("init upload: %w", err)
	}
	if err := json.NewDecoder(resp.Body).Decode(&up); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("decode: %w", err)
	}
	resp.Body.Close()

	if up.Response.FileURL != "" {
		return &provider.Result{OutputURL: up.Response.FileURL}, nil
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := fw.Write(fileData); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	w.Close()

	ureq, err := http.NewRequestWithContext(ctx, "POST", up.Response.UploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}
	ureq.Header.Set("Content-Type", w.FormDataContentType())

	var up2 uploadResp
	resp2, err := p.client.Do(ureq)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}
	defer resp2.Body.Close()

	if err := json.NewDecoder(resp2.Body).Decode(&up2); err != nil {
		return nil, fmt.Errorf("decode upload: %w", err)
	}

	return &provider.Result{
		OutputURL: up2.Response.FileURL,
		FileCode:  up2.Response.UploadID,
		FileSize:  fileSize,
	}, nil
}

func (p *RapidGator) UploadFromURL(ctx context.Context, sourceURL string, opts map[string]string) (*provider.Result, error) {
	token, err := p.getToken(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	u := fmt.Sprintf("%s/remote/create?token=%s&url=%s", apiBase, token, url.QueryEscape(sourceURL))
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
		FileCode: fmt.Sprintf("%d", r.Response.ID),
	}, nil
}

func (p *RapidGator) getToken(ctx context.Context, opts map[string]string) (string, error) {
	if token, ok := opts["token"]; ok && token != "" {
		return token, nil
	}

	username := opts["username"]
	password := opts["password"]
	if username == "" || password == "" {
		return "", fmt.Errorf("rapidgator needs token or username/password in options")
	}

	u := fmt.Sprintf("%s/user/login?username=%s&password=%s",
		apiBase, url.QueryEscape(username), url.QueryEscape(password))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}

	var l loginResp
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
		return "", err
	}

	return l.Response.Token, nil
}
