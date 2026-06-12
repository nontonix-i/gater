package provider

import (
	"context"
	"io"
)

type ProgressFunc func(percentage int, msg string)

type progressKey struct{}

func WithProgress(ctx context.Context, fn ProgressFunc) context.Context {
	return context.WithValue(ctx, progressKey{}, fn)
}

func GetProgress(ctx context.Context) (ProgressFunc, bool) {
	fn, ok := ctx.Value(progressKey{}).(ProgressFunc)
	return fn, ok
}

type ProgressReader struct {
	reader io.Reader
	total  int64
	read   int64
	fn     ProgressFunc
}

func NewProgressReader(r io.Reader, total int64, fn ProgressFunc) *ProgressReader {
	return &ProgressReader{reader: r, total: total, read: 0, fn: fn}
}

func (r *ProgressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)
	if r.total > 0 {
		pct := int(r.read * 100 / r.total)
		r.fn(pct, "uploading")
	}
	return n, err
}
