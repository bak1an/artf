package cleanup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
	"github.com/bak1an/artf/store/mock"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunOnce_NoRepos(t *testing.T) {
	repos := &mock.MockRepoStore{
		ListFn: func(ctx context.Context) ([]*store.Repo, error) {
			return nil, nil
		},
	}
	artifacts := &mock.MockArtifactStore{}

	c := New(repos, artifacts, "/tmp/test", testLogger())

	if err := c.RunOnce(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunOnce_BothLimitsZero(t *testing.T) {
	repos := &mock.MockRepoStore{
		ListFn: func(ctx context.Context) ([]*store.Repo, error) {
			return []*store.Repo{
				{ID: 1, Name: "r1", KeepCount: 0, KeepDays: 0},
			}, nil
		},
	}

	listByRepoCalled := false
	artifacts := &mock.MockArtifactStore{
		ListByRepoFn: func(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
			listByRepoCalled = true
			return nil, nil
		},
	}

	c := New(repos, artifacts, "/tmp/test", testLogger())

	if err := c.RunOnce(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if listByRepoCalled {
		t.Fatal("ListByRepo should not be called when both limits are 0")
	}
}

func TestRunOnce_KeepCount(t *testing.T) {
	now := time.Now()
	arts := make([]*store.Artifact, 5)
	for i := range arts {
		arts[i] = &store.Artifact{
			ID:        uint64(i + 1),
			RepoID:    1,
			Path:      fmt.Sprintf("repos/r1/art%d", i+1),
			CreatedAt: now.Add(-time.Duration(i) * time.Hour), // 0 is newest
		}
	}

	repos := &mock.MockRepoStore{
		ListFn: func(ctx context.Context) ([]*store.Repo, error) {
			return []*store.Repo{
				{ID: 1, Name: "r1", KeepCount: 2},
			}, nil
		},
	}

	var removedFiles []string
	var deletedIDs []uint64

	artifacts := &mock.MockArtifactStore{
		ListByRepoFn: func(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
			return arts, nil
		},
		DeleteFn: func(ctx context.Context, id uint64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	}

	c := New(repos, artifacts, "/data", testLogger())
	c.removeFile = func(path string) error {
		removedFiles = append(removedFiles, path)
		return nil
	}

	if err := c.RunOnce(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deletedIDs) != 3 {
		t.Fatalf("expected 3 deletions, got %d", len(deletedIDs))
	}
	// The 3 oldest (IDs 3, 4, 5) should be deleted
	for _, id := range deletedIDs {
		if id < 3 {
			t.Errorf("unexpected deletion of artifact %d (should keep newest 2)", id)
		}
	}
	if len(removedFiles) != 3 {
		t.Fatalf("expected 3 file removals, got %d", len(removedFiles))
	}
}

func TestRunOnce_KeepDays(t *testing.T) {
	now := time.Now()
	arts := []*store.Artifact{
		{ID: 1, RepoID: 1, Path: "repos/r1/a1", CreatedAt: now.Add(-1 * 24 * time.Hour)},  // 1 day old
		{ID: 2, RepoID: 1, Path: "repos/r1/a2", CreatedAt: now.Add(-5 * 24 * time.Hour)},  // 5 days old
		{ID: 3, RepoID: 1, Path: "repos/r1/a3", CreatedAt: now.Add(-10 * 24 * time.Hour)}, // 10 days old
		{ID: 4, RepoID: 1, Path: "repos/r1/a4", CreatedAt: now.Add(-20 * 24 * time.Hour)}, // 20 days old
	}

	repos := &mock.MockRepoStore{
		ListFn: func(ctx context.Context) ([]*store.Repo, error) {
			return []*store.Repo{
				{ID: 1, Name: "r1", KeepDays: 7},
			}, nil
		},
	}

	var deletedIDs []uint64

	artifacts := &mock.MockArtifactStore{
		ListByRepoFn: func(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
			return arts, nil
		},
		DeleteFn: func(ctx context.Context, id uint64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	}

	c := New(repos, artifacts, "/data", testLogger())
	c.removeFile = func(path string) error { return nil }

	if err := c.RunOnce(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Artifacts 3 and 4 are older than 7 days
	if len(deletedIDs) != 2 {
		t.Fatalf("expected 2 deletions, got %d: %v", len(deletedIDs), deletedIDs)
	}
}

func TestRunOnce_KeepDaysAllExpired(t *testing.T) {
	now := time.Now()
	arts := []*store.Artifact{
		{ID: 1, RepoID: 1, Path: "repos/r1/a1", CreatedAt: now.Add(-10 * 24 * time.Hour)},
		{ID: 2, RepoID: 1, Path: "repos/r1/a2", CreatedAt: now.Add(-20 * 24 * time.Hour)},
		{ID: 3, RepoID: 1, Path: "repos/r1/a3", CreatedAt: now.Add(-30 * 24 * time.Hour)},
	}

	repos := &mock.MockRepoStore{
		ListFn: func(ctx context.Context) ([]*store.Repo, error) {
			return []*store.Repo{
				{ID: 1, Name: "r1", KeepDays: 7},
			}, nil
		},
	}

	var deletedIDs []uint64

	artifacts := &mock.MockArtifactStore{
		ListByRepoFn: func(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
			return arts, nil
		},
		DeleteFn: func(ctx context.Context, id uint64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	}

	c := New(repos, artifacts, "/data", testLogger())
	c.removeFile = func(path string) error { return nil }

	if err := c.RunOnce(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All expired, but keep newest 1 (ID=1), delete 2 and 3
	if len(deletedIDs) != 2 {
		t.Fatalf("expected 2 deletions, got %d: %v", len(deletedIDs), deletedIDs)
	}
	for _, id := range deletedIDs {
		if id == 1 {
			t.Error("should not delete newest artifact even if expired")
		}
	}
}

func TestRunOnce_BothSet_KeepCountGoverns(t *testing.T) {
	now := time.Now()
	arts := make([]*store.Artifact, 5)
	for i := range arts {
		arts[i] = &store.Artifact{
			ID:        uint64(i + 1),
			RepoID:    1,
			Path:      fmt.Sprintf("repos/r1/art%d", i+1),
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		}
	}

	repos := &mock.MockRepoStore{
		ListFn: func(ctx context.Context) ([]*store.Repo, error) {
			return []*store.Repo{
				{ID: 1, Name: "r1", KeepCount: 3, KeepDays: 1},
			}, nil
		},
	}

	var deletedIDs []uint64

	artifacts := &mock.MockArtifactStore{
		ListByRepoFn: func(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
			return arts, nil
		},
		DeleteFn: func(ctx context.Context, id uint64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	}

	c := New(repos, artifacts, "/data", testLogger())
	c.removeFile = func(path string) error { return nil }

	if err := c.RunOnce(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// KeepCount=3 governs: keep 3 newest, delete 2
	if len(deletedIDs) != 2 {
		t.Fatalf("expected 2 deletions, got %d: %v", len(deletedIDs), deletedIDs)
	}
}

func TestRunOnce_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	repos := &mock.MockRepoStore{
		ListFn: func(ctx context.Context) ([]*store.Repo, error) {
			return []*store.Repo{
				{ID: 1, Name: "r1", KeepCount: 1},
				{ID: 2, Name: "r2", KeepCount: 1},
			}, nil
		},
	}

	var mu sync.Mutex
	processedRepos := 0

	now := time.Now()
	artifacts := &mock.MockArtifactStore{
		ListByRepoFn: func(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
			mu.Lock()
			processedRepos++
			mu.Unlock()
			// Cancel after first repo
			cancel()
			return []*store.Artifact{
				{ID: repoID*10 + 1, RepoID: repoID, Path: "p1", CreatedAt: now},
			}, nil
		},
		DeleteFn: func(ctx context.Context, id uint64) error {
			return nil
		},
	}

	c := New(repos, artifacts, "/data", testLogger())
	c.removeFile = func(path string) error { return nil }

	err := c.RunOnce(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if processedRepos > 1 {
		t.Fatalf("expected at most 1 repo processed, got %d", processedRepos)
	}
}

func TestRunOnce_FileMissing(t *testing.T) {
	now := time.Now()
	arts := []*store.Artifact{
		{ID: 1, RepoID: 1, Path: "repos/r1/a1", CreatedAt: now},
		{ID: 2, RepoID: 1, Path: "repos/r1/a2", CreatedAt: now.Add(-time.Hour)},
	}

	repos := &mock.MockRepoStore{
		ListFn: func(ctx context.Context) ([]*store.Repo, error) {
			return []*store.Repo{
				{ID: 1, Name: "r1", KeepCount: 1},
			}, nil
		},
	}

	var deletedIDs []uint64
	artifacts := &mock.MockArtifactStore{
		ListByRepoFn: func(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
			return arts, nil
		},
		DeleteFn: func(ctx context.Context, id uint64) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	}

	c := New(repos, artifacts, "/data", testLogger())
	c.removeFile = func(path string) error {
		return fmt.Errorf("remove %s: %w", path, os.ErrNotExist)
	}

	if err := c.RunOnce(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still delete the DB record even though file is missing
	if len(deletedIDs) != 1 {
		t.Fatalf("expected 1 deletion, got %d", len(deletedIDs))
	}
}
