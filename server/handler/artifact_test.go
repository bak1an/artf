package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
	"github.com/bak1an/artf/store/mock"
)

func TestListArtifacts(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("success", func(t *testing.T) {
		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, name string) (*store.Repo, error) {
				return &store.Repo{ID: 1, Name: name}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			ListByRepoFn: func(_ context.Context, repoID uint64) ([]*store.Artifact, error) {
				return []*store.Artifact{
					{ID: 1, Name: "v1.tar.gz", CreatedAt: now},
					{ID: 2, Name: "v2.tar.gz", CreatedAt: now},
				}, nil
			},
		}

		r := httptest.NewRequest("GET", "/myrepo", nil)
		r.SetPathValue("repoName", "myrepo")
		w := httptest.NewRecorder()

		ListArtifacts(repos, artifacts)(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var result []artifactInfo
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 artifacts, got %d", len(result))
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return nil, store.ErrNotFound
			},
		}

		r := httptest.NewRequest("GET", "/missing", nil)
		r.SetPathValue("repoName", "missing")
		w := httptest.NewRecorder()

		ListArtifacts(repos, &mock.MockArtifactStore{})(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return &store.Repo{ID: 1}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			ListByRepoFn: func(_ context.Context, _ uint64) ([]*store.Artifact, error) {
				return nil, errors.New("db error")
			},
		}

		r := httptest.NewRequest("GET", "/myrepo", nil)
		r.SetPathValue("repoName", "myrepo")
		w := httptest.NewRecorder()

		ListArtifacts(repos, artifacts)(w, r)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestDownloadArtifact(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dataDir := t.TempDir()
		repoDir := filepath.Join(dataDir, "repos", "myrepo")
		os.MkdirAll(repoDir, 0o755)
		os.WriteFile(filepath.Join(repoDir, "v1.tar.gz"), []byte("file-content"), 0o644)

		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return &store.Repo{ID: 1, Name: "myrepo"}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			GetByRepoAndNameFn: func(_ context.Context, _ uint64, name string) (*store.Artifact, error) {
				return &store.Artifact{ID: 1, Name: name, Path: "repos/myrepo/v1.tar.gz"}, nil
			},
		}

		r := httptest.NewRequest("GET", "/myrepo/v1.tar.gz", nil)
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "v1.tar.gz")
		w := httptest.NewRecorder()

		DownloadArtifact(repos, artifacts, dataDir)(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if w.Body.String() != "file-content" {
			t.Fatalf("unexpected body: %q", w.Body.String())
		}
	})

	t.Run("latest resolves", func(t *testing.T) {
		dataDir := t.TempDir()
		repoDir := filepath.Join(dataDir, "repos", "myrepo")
		os.MkdirAll(repoDir, 0o755)
		os.WriteFile(filepath.Join(repoDir, "v2.tar.gz"), []byte("latest-content"), 0o644)

		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return &store.Repo{ID: 1, Name: "myrepo"}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			GetLatestByRepoFn: func(_ context.Context, _ uint64) (*store.Artifact, error) {
				return &store.Artifact{ID: 2, Name: "v2.tar.gz", Path: "repos/myrepo/v2.tar.gz"}, nil
			},
		}

		r := httptest.NewRequest("GET", "/myrepo/latest", nil)
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "latest")
		w := httptest.NewRecorder()

		DownloadArtifact(repos, artifacts, dataDir)(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if w.Body.String() != "latest-content" {
			t.Fatalf("unexpected body: %q", w.Body.String())
		}
	})

	t.Run("artifact not found", func(t *testing.T) {
		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return &store.Repo{ID: 1}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			GetByRepoAndNameFn: func(_ context.Context, _ uint64, _ string) (*store.Artifact, error) {
				return nil, store.ErrNotFound
			},
		}

		r := httptest.NewRequest("GET", "/myrepo/missing", nil)
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "missing")
		w := httptest.NewRecorder()

		DownloadArtifact(repos, artifacts, t.TempDir())(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return nil, store.ErrNotFound
			},
		}

		r := httptest.NewRequest("GET", "/missing/v1", nil)
		r.SetPathValue("repoName", "missing")
		r.SetPathValue("artifactName", "v1")
		w := httptest.NewRecorder()

		DownloadArtifact(repos, &mock.MockArtifactStore{}, t.TempDir())(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestUploadArtifact(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dataDir := t.TempDir()
		repoDir := filepath.Join(dataDir, "repos", "myrepo")
		os.MkdirAll(repoDir, 0o755)

		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return &store.Repo{ID: 1, Name: "myrepo", Path: "repos/myrepo"}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			GetByRepoAndNameFn: func(_ context.Context, _ uint64, _ string) (*store.Artifact, error) {
				return nil, store.ErrNotFound
			},
			CreateFn: func(_ context.Context, a *store.Artifact) error {
				a.ID = 1
				return nil
			},
		}

		r := httptest.NewRequest("PUT", "/myrepo/v1.tar.gz", strings.NewReader("upload-data"))
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "v1.tar.gz")
		w := httptest.NewRecorder()

		UploadArtifact(repos, artifacts, dataDir, 1024)(w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		data, err := os.ReadFile(filepath.Join(repoDir, "v1.tar.gz"))
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if string(data) != "upload-data" {
			t.Fatalf("unexpected file content: %q", string(data))
		}

		var result artifactInfo
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if result.Name != "v1.tar.gz" {
			t.Fatalf("unexpected name: %q", result.Name)
		}
	})

	t.Run("reserved name latest", func(t *testing.T) {
		r := httptest.NewRequest("PUT", "/myrepo/latest", strings.NewReader("data"))
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "latest")
		w := httptest.NewRecorder()

		UploadArtifact(&mock.MockRepoStore{}, &mock.MockArtifactStore{}, t.TempDir(), 1024)(w, r)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		r := httptest.NewRequest("PUT", "/myrepo/bad%20name", strings.NewReader("data"))
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "bad name")
		w := httptest.NewRecorder()

		UploadArtifact(&mock.MockRepoStore{}, &mock.MockArtifactStore{}, t.TempDir(), 1024)(w, r)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return nil, store.ErrNotFound
			},
		}

		r := httptest.NewRequest("PUT", "/missing/v1.tar.gz", strings.NewReader("data"))
		r.SetPathValue("repoName", "missing")
		r.SetPathValue("artifactName", "v1.tar.gz")
		w := httptest.NewRecorder()

		UploadArtifact(repos, &mock.MockArtifactStore{}, t.TempDir(), 1024)(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("db error cleans up file", func(t *testing.T) {
		dataDir := t.TempDir()
		repoDir := filepath.Join(dataDir, "repos", "myrepo")
		os.MkdirAll(repoDir, 0o755)

		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return &store.Repo{ID: 1, Name: "myrepo", Path: "repos/myrepo"}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			GetByRepoAndNameFn: func(_ context.Context, _ uint64, _ string) (*store.Artifact, error) {
				return nil, store.ErrNotFound
			},
			CreateFn: func(_ context.Context, _ *store.Artifact) error {
				return errors.New("db error")
			},
		}

		r := httptest.NewRequest("PUT", "/myrepo/v1.tar.gz", strings.NewReader("data"))
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "v1.tar.gz")
		w := httptest.NewRecorder()

		UploadArtifact(repos, artifacts, dataDir, 1024)(w, r)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}

		if _, err := os.Stat(filepath.Join(repoDir, "v1.tar.gz")); !os.IsNotExist(err) {
			t.Fatal("expected file to be cleaned up after db error")
		}
	})

	t.Run("duplicate upload returns conflict without overwrite", func(t *testing.T) {
		dataDir := t.TempDir()
		repoDir := filepath.Join(dataDir, "repos", "myrepo")
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(repoDir, "v1.tar.gz"), []byte("existing-data"), 0o644); err != nil {
			t.Fatalf("write existing file: %v", err)
		}

		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return &store.Repo{ID: 1, Name: "myrepo", Path: "repos/myrepo"}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			GetByRepoAndNameFn: func(_ context.Context, _ uint64, _ string) (*store.Artifact, error) {
				return &store.Artifact{ID: 1, Name: "v1.tar.gz"}, nil
			},
			CreateFn: func(_ context.Context, _ *store.Artifact) error {
				t.Fatal("Create should not be called for duplicate uploads")
				return nil
			},
		}

		r := httptest.NewRequest("PUT", "/myrepo/v1.tar.gz", strings.NewReader("new-data"))
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "v1.tar.gz")
		w := httptest.NewRecorder()

		UploadArtifact(repos, artifacts, dataDir, 1024)(w, r)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}

		data, err := os.ReadFile(filepath.Join(repoDir, "v1.tar.gz"))
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if string(data) != "existing-data" {
			t.Fatalf("unexpected file content: %q", string(data))
		}
	})

	t.Run("artifact existence check error", func(t *testing.T) {
		dataDir := t.TempDir()
		repoDir := filepath.Join(dataDir, "repos", "myrepo")
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return &store.Repo{ID: 1, Name: "myrepo", Path: "repos/myrepo"}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			GetByRepoAndNameFn: func(_ context.Context, _ uint64, _ string) (*store.Artifact, error) {
				return nil, errors.New("db error")
			},
		}

		r := httptest.NewRequest("PUT", "/myrepo/v1.tar.gz", strings.NewReader("data"))
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "v1.tar.gz")
		w := httptest.NewRecorder()

		UploadArtifact(repos, artifacts, dataDir, 1024)(w, r)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("request too large", func(t *testing.T) {
		dataDir := t.TempDir()
		repoDir := filepath.Join(dataDir, "repos", "myrepo")
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		repos := &mock.MockRepoStore{
			GetByNameFn: func(_ context.Context, _ string) (*store.Repo, error) {
				return &store.Repo{ID: 1, Name: "myrepo", Path: "repos/myrepo"}, nil
			},
		}
		artifacts := &mock.MockArtifactStore{
			GetByRepoAndNameFn: func(_ context.Context, _ uint64, _ string) (*store.Artifact, error) {
				return nil, store.ErrNotFound
			},
			CreateFn: func(_ context.Context, _ *store.Artifact) error {
				t.Fatal("Create should not be called for oversized uploads")
				return nil
			},
		}

		r := httptest.NewRequest("PUT", "/myrepo/v1.tar.gz", strings.NewReader("12345"))
		r.SetPathValue("repoName", "myrepo")
		r.SetPathValue("artifactName", "v1.tar.gz")
		w := httptest.NewRecorder()

		UploadArtifact(repos, artifacts, dataDir, 4)(w, r)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413, got %d", w.Code)
		}

		if _, err := os.Stat(filepath.Join(repoDir, "v1.tar.gz")); !os.IsNotExist(err) {
			t.Fatal("expected final file to not exist after oversized upload")
		}

		entries, err := os.ReadDir(repoDir)
		if err != nil {
			t.Fatalf("read dir: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected temp files to be cleaned up, found %d entries", len(entries))
		}
	})
}
