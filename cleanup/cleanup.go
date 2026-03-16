package cleanup

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/bak1an/artf/store"
)

// Cleaner periodically removes stale artifacts based on repo retention policies.
type Cleaner struct {
	repos      store.RepoStore
	artifacts  store.ArtifactStore
	dataDir    string
	logger     *slog.Logger
	interval   time.Duration
	removeFile func(string) error
}

// New creates a new Cleaner with a 24-hour default interval.
func New(repos store.RepoStore, artifacts store.ArtifactStore, dataDir string, logger *slog.Logger) *Cleaner {
	return &Cleaner{
		repos:      repos,
		artifacts:  artifacts,
		dataDir:    dataDir,
		logger:     logger,
		interval:   24 * time.Hour,
		removeFile: os.Remove,
	}
}

// Run runs cleanup immediately, then on a ticker. Returns when ctx is cancelled.
func (c *Cleaner) Run(ctx context.Context) {
	if err := c.RunOnce(ctx); err != nil {
		c.logger.Error("cleanup pass failed", "error", err)
	}

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.RunOnce(ctx); err != nil {
				c.logger.Error("cleanup pass failed", "error", err)
			}
		}
	}
}

// RunOnce performs a single cleanup pass over all repos.
func (c *Cleaner) RunOnce(ctx context.Context) error {
	repos, err := c.repos.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()

	for _, repo := range repos {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if repo.KeepCount == 0 && repo.KeepDays == 0 {
			continue
		}

		deleted, err := c.cleanRepo(ctx, repo, now)
		if err != nil {
			c.logger.Error("cleanup failed for repo", "repo", repo.Name, "error", err)
			continue
		}
		if deleted > 0 {
			c.logger.Info("cleaned up artifacts", "repo", repo.Name, "deleted", deleted)
		}
	}

	return nil
}

func (c *Cleaner) cleanRepo(ctx context.Context, repo *store.Repo, now time.Time) (int, error) {
	artifacts, err := c.artifacts.ListByRepo(ctx, repo.ID)
	if err != nil {
		return 0, err
	}

	if len(artifacts) == 0 {
		return 0, nil
	}

	// Sort by CreatedAt descending (newest first)
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].CreatedAt.After(artifacts[j].CreatedAt)
	})

	var toDelete []*store.Artifact

	if repo.KeepCount > 0 {
		// KeepCount governs (regardless of KeepDays)
		if len(artifacts) > repo.KeepCount {
			toDelete = artifacts[repo.KeepCount:]
		}
	} else {
		// KeepCount == 0, KeepDays > 0
		cutoff := now.AddDate(0, 0, -repo.KeepDays)
		for i, a := range artifacts {
			if a.CreatedAt.Before(cutoff) {
				// Always keep at least 1 (the newest)
				if i == 0 {
					continue
				}
				toDelete = append(toDelete, a)
			}
		}
	}

	deleted := 0
	for _, a := range toDelete {
		filePath := filepath.Join(c.dataDir, a.Path)
		if err := c.removeFile(filePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				c.logger.Warn("artifact file already missing", "path", filePath, "artifact_id", a.ID)
			} else {
				c.logger.Error("failed to remove artifact file", "path", filePath, "error", err)
				continue
			}
		}

		if err := c.artifacts.Delete(ctx, a.ID); err != nil {
			c.logger.Error("failed to delete artifact record", "artifact_id", a.ID, "error", err)
			continue
		}

		deleted++
	}

	return deleted, nil
}
