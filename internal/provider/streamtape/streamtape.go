package streamtape

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

type Streamtape struct {
	client *http.Client
}

type ulResult struct {
	Result struct {
		URL string `json:"url"`
		FN  string `json:"fn"`
	} `json:"result"`
}

type uploadResp struct {
	Result struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	} `json:"result"`
}

type remoteResp struct {
	Result []struct {
		URL  string `json:"url"`
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"result"`
}

func New() *Streamtape {
	return &Streamtape{
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *Streamtape) Name() string             { return "streamtape" }
func (p *Streamtape) Type() provider.Type       { return provider.TypeVideoHost }
func (p *Streamtape) SupportsAnonymous() bool   { return false }
func (p *Streamtape) SupportsRemoteURL() bool   { return true }
func (p *Streamtape) HasAPI() bool              { return true }

func (p *Streamtape) Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*provider.Result, error) {
	login, key, err := p.getCreds(opts)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("https://api.streamtape.com/file/ul?login=%s&key=%s", login, key)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("get upload url: %w", err)
	}

	var ul ulResult
	if err := p.doJSON(req, &ul); err != nil {
		return nil, fmt.Errorf("get upload url: %w", err)
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	fw, err := w.CreateFormFile("file1", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, file); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}
	w.Close()

	ureq, err := http.NewRequestWithContext(ctx, "POST", ul.Result.URL, body)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}
	ureq.Header.Set("Content-Type", w.FormDataContentType())

	var up uploadResp
	if err := p.doJSON(ureq, &up); err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}

	return &provider.Result{
		OutputURL: up.Result.URL,
		FileName:  up.Result.Name,
	}, nil
}

func (p *Streamtape) UploadFromURL(ctx context.Context, urlStr string, opts map[string]string) (*provider.Result, error) {
	login, key, err := p.getCreds(opts)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("https://api.streamtape.com/remotedl/add?login=%s&key=%s&url=%s",
		login, key, url.QueryEscape(urlStr))

	if name, ok := opts["name"]; ok {
		u += "&name=" + url.QueryEscape(name)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("create remote request: %w", err)
	}

	var resp remoteResp
	if err := p.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("remote upload: %w", err)
	}

	if len(resp.Result) == 0 {
		return nil, fmt.Errorf("no result from remote upload")
	}

	return &provider.Result{
		OutputURL: resp.Result[0].URL,
		FileCode:  resp.Result[0].ID,
		FileName:  resp.Result[0].Name,
	}, nil
}

func (p *Streamtape) getCreds(opts map[string]string) (login, key string, err error) {
	login = opts["login"]
	key = opts["key"]
	if login == "" || key == "" {
		return "", "", fmt.Errorf("streamtape requires login and key in options")
	}
	return
}

func (p *Streamtape) doJSON(req *http.Request, v interface{}) error {
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(v)
}
