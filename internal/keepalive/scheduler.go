package keepalive

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/user/gater/config"
	"github.com/user/gater/internal/model"
)

type Scheduler struct {
	db     *gorm.DB
	client *http.Client
	cfg    config.KeepaliveConfig
	stopCh chan struct{}
}

func New(db *gorm.DB, cfg config.KeepaliveConfig) *Scheduler {
	return &Scheduler{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	if !s.cfg.Enabled {
		slog.Info("keepalive scheduler disabled")
		return
	}

	interval := time.Duration(s.cfg.CheckEvery) * time.Minute
	slog.Info("keepalive scheduler started",
		"check_every", s.cfg.CheckEvery,
		"visit_older", s.cfg.VisitOlder,
		"request_limit", s.cfg.RequestLimit,
	)

	s.runOnce(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx)
		case <-s.stopCh:
			slog.Info("keepalive scheduler stopped")
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) runOnce(ctx context.Context) {
	slog.Debug("keepalive check starting")
	cutoff := time.Now().AddDate(0, 0, -s.cfg.VisitOlder)

	var results []model.TaskResult
	err := s.db.Where(
		"status = ? AND output_url != '' AND (last_keepalive_at IS NULL OR last_keepalive_at < ?)",
		"completed", cutoff,
	).Order("last_keepalive_at ASC NULLS FIRST").
		Limit(s.cfg.RequestLimit).
		Find(&results).Error
	if err != nil {
		slog.Error("keepalive query failed", "error", err)
		return
	}

	if len(results) == 0 {
		slog.Debug("keepalive no URLs to visit")
		return
	}

	slog.Info("keepalive visiting URLs", "count", len(results))

	for _, r := range results {
		select {
		case <-ctx.Done():
			return
		default:
		}

		s.visit(ctx, &r)
	}
}

func (s *Scheduler) visit(ctx context.Context, r *model.TaskResult) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.OutputURL, nil)
	if err != nil {
		slog.Error("keepalive request creation failed",
			"provider", r.Provider,
			"output_url", r.OutputURL,
			"error", err,
		)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		slog.Error("keepalive request failed",
			"provider", r.Provider,
			"output_url", r.OutputURL,
			"error", err,
		)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		now := time.Now()
		err := s.db.Model(r).Updates(map[string]interface{}{
			"last_keepalive_at": now,
			"keepalive_count":   r.KeepaliveCount + 1,
		}).Error
		if err != nil {
			slog.Error("keepalive update failed",
				"provider", r.Provider,
				"error", err,
			)
		} else {
			slog.Debug("keepalive success",
				"provider", r.Provider,
				"output_url", r.OutputURL,
				"status", resp.StatusCode,
			)
		}
	} else {
		slog.Warn("keepalive unexpected status",
			"provider", r.Provider,
			"output_url", r.OutputURL,
			"status", resp.StatusCode,
		)
	}
}
