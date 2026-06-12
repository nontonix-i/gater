package provider

import (
	"context"
	"io"
)

type Type string

const (
	TypeVideoHost Type = "video_host"
	TypeStorage   Type = "storage"
	TypeBoth      Type = "both"
)

type Result struct {
	OutputURL string
	FileCode  string
	FileName  string
	FileSize  int64
}

type Provider interface {
	Name() string
	Type() Type
	SupportsAnonymous() bool
	SupportsRemoteURL() bool
	HasAPI() bool
	Upload(ctx context.Context, file io.Reader, filename string, opts map[string]string) (*Result, error)
	UploadFromURL(ctx context.Context, url string, opts map[string]string) (*Result, error)
}
