package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
	"github.com/bak1an/artf/store/mock"
)

func TestGetRepoInfoHandler(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	t.Run("success", func(t *testing.T) {
		rs := &mock.MockRepoStore{
			GetFn: func(_ context.Context, id uint64) (*store.Repo, error) {
				if id != 7 {
					t.Fatalf("unexpected id: %d", id)
				}
				return &store.Repo{
					ID:        7,
					Name:      "my-repo",
					Type:      store.RepoTypeFile,
					Path:      "repos/my-repo",
					KeepCount: 5,
					KeepDays:  30,
					CreatedAt: now.Add(-time.Hour),
					UpdatedAt: now,
				}, nil
			},
		}
		as := &mock.MockArtifactStore{
			ListByRepoFn: func(_ context.Context, repoID uint64) ([]*store.Artifact, error) {
				if repoID != 7 {
					t.Fatalf("unexpected repo id: %d", repoID)
				}
				return []*store.Artifact{
					{
						ID:        11,
						Name:      "v1.0.0.tar.gz",
						RepoID:    7,
						Path:      "repos/my-repo/v1.0.0.tar.gz",
						CreatedAt: now.Add(-2 * time.Minute),
					},
					{
						ID:        12,
						Name:      "v1.0.1.tar.gz",
						RepoID:    7,
						Path:      "repos/my-repo/v1.0.1.tar.gz",
						CreatedAt: now.Add(-time.Minute),
					},
				}, nil
			},
		}

		h := getRepoInfoHandler(rs, as)
		req := httptest.NewRequest(http.MethodGet, "/repos/7", nil)
		req.SetPathValue("id", "7")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp RepoInfoResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if resp.Repo.ID != 7 {
			t.Fatalf("expected repo id 7, got %d", resp.Repo.ID)
		}
		if resp.Repo.Name != "my-repo" {
			t.Fatalf("expected repo name my-repo, got %s", resp.Repo.Name)
		}
		if resp.Repo.Type != string(store.RepoTypeFile) {
			t.Fatalf("expected repo type %s, got %s", store.RepoTypeFile, resp.Repo.Type)
		}
		if resp.Repo.Path != "repos/my-repo" {
			t.Fatalf("expected repo path repos/my-repo, got %s", resp.Repo.Path)
		}
		if resp.Repo.KeepCount != 5 {
			t.Fatalf("expected keep count 5, got %d", resp.Repo.KeepCount)
		}
		if resp.Repo.KeepDays != 30 {
			t.Fatalf("expected keep days 30, got %d", resp.Repo.KeepDays)
		}
		if !resp.Repo.CreatedAt.Equal(now.Add(-time.Hour)) {
			t.Fatalf("unexpected created at: %s", resp.Repo.CreatedAt)
		}
		if !resp.Repo.UpdatedAt.Equal(now) {
			t.Fatalf("unexpected updated at: %s", resp.Repo.UpdatedAt)
		}
		if resp.Repo.ArtifactCount != 2 {
			t.Fatalf("expected artifact count 2, got %d", resp.Repo.ArtifactCount)
		}

		if len(resp.Artifacts) != 2 {
			t.Fatalf("expected 2 artifacts, got %d", len(resp.Artifacts))
		}
		if resp.Artifacts[0].ID != 11 || resp.Artifacts[1].ID != 12 {
			t.Fatalf("unexpected artifact ids: %+v", resp.Artifacts)
		}
	})

	t.Run("bad id", func(t *testing.T) {
		h := getRepoInfoHandler(&mock.MockRepoStore{}, &mock.MockArtifactStore{})
		req := httptest.NewRequest(http.MethodGet, "/repos/x", nil)
		req.SetPathValue("id", "x")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		rs := &mock.MockRepoStore{
			GetFn: func(_ context.Context, _ uint64) (*store.Repo, error) {
				return nil, store.ErrNotFound
			},
		}
		h := getRepoInfoHandler(rs, &mock.MockArtifactStore{})
		req := httptest.NewRequest(http.MethodGet, "/repos/9", nil)
		req.SetPathValue("id", "9")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("repo get error", func(t *testing.T) {
		rs := &mock.MockRepoStore{
			GetFn: func(_ context.Context, _ uint64) (*store.Repo, error) {
				return nil, errors.New("db read failed")
			},
		}
		h := getRepoInfoHandler(rs, &mock.MockArtifactStore{})
		req := httptest.NewRequest(http.MethodGet, "/repos/9", nil)
		req.SetPathValue("id", "9")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})

	t.Run("artifact list error", func(t *testing.T) {
		rs := &mock.MockRepoStore{
			GetFn: func(_ context.Context, id uint64) (*store.Repo, error) {
				return &store.Repo{
					ID:        id,
					Name:      "x",
					Type:      store.RepoTypeFile,
					Path:      "repos/x",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
		}
		as := &mock.MockArtifactStore{
			ListByRepoFn: func(_ context.Context, _ uint64) ([]*store.Artifact, error) {
				return nil, errors.New("db list failed")
			},
		}
		h := getRepoInfoHandler(rs, as)
		req := httptest.NewRequest(http.MethodGet, "/repos/9", nil)
		req.SetPathValue("id", "9")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})
}

func TestUpdateRepoHandler(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	t.Run("success update both fields", func(t *testing.T) {
		var updatedRepo *store.Repo
		rs := &mock.MockRepoStore{
			GetFn: func(_ context.Context, id uint64) (*store.Repo, error) {
				if id != 7 {
					t.Fatalf("unexpected id: %d", id)
				}
				return &store.Repo{
					ID:        7,
					Name:      "my-repo",
					Type:      store.RepoTypeFile,
					Path:      "repos/my-repo",
					KeepCount: 5,
					KeepDays:  30,
					CreatedAt: now.Add(-time.Hour),
					UpdatedAt: now.Add(-time.Hour),
				}, nil
			},
			UpdateFn: func(_ context.Context, repo *store.Repo) error {
				updatedRepo = repo
				return nil
			},
		}

		keepCount := 10
		keepDays := 60
		body, _ := json.Marshal(RepoUpdateRequest{KeepCount: &keepCount, KeepDays: &keepDays})
		req := httptest.NewRequest(http.MethodPatch, "/repos/7", bytes.NewReader(body))
		req.SetPathValue("id", "7")
		rr := httptest.NewRecorder()
		h := updateRepoHandler(rs)
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		if updatedRepo == nil {
			t.Fatal("Update was not called")
		}
		if updatedRepo.KeepCount != 10 {
			t.Fatalf("expected keep count 10, got %d", updatedRepo.KeepCount)
		}
		if updatedRepo.KeepDays != 60 {
			t.Fatalf("expected keep days 60, got %d", updatedRepo.KeepDays)
		}
		if updatedRepo.UpdatedAt.Before(now) {
			t.Fatalf("expected UpdatedAt to be updated, got %s", updatedRepo.UpdatedAt)
		}

		var resp Repo
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.KeepCount != 10 || resp.KeepDays != 60 {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("partial update keep_count only", func(t *testing.T) {
		var updatedRepo *store.Repo
		rs := &mock.MockRepoStore{
			GetFn: func(_ context.Context, id uint64) (*store.Repo, error) {
				return &store.Repo{
					ID:        7,
					Name:      "my-repo",
					Type:      store.RepoTypeFile,
					Path:      "repos/my-repo",
					KeepCount: 5,
					KeepDays:  30,
					CreatedAt: now.Add(-time.Hour),
					UpdatedAt: now.Add(-time.Hour),
				}, nil
			},
			UpdateFn: func(_ context.Context, repo *store.Repo) error {
				updatedRepo = repo
				return nil
			},
		}

		keepCount := 20
		body, _ := json.Marshal(RepoUpdateRequest{KeepCount: &keepCount})
		req := httptest.NewRequest(http.MethodPatch, "/repos/7", bytes.NewReader(body))
		req.SetPathValue("id", "7")
		rr := httptest.NewRecorder()
		h := updateRepoHandler(rs)
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		if updatedRepo.KeepCount != 20 {
			t.Fatalf("expected keep count 20, got %d", updatedRepo.KeepCount)
		}
		if updatedRepo.KeepDays != 30 {
			t.Fatalf("expected keep days to remain 30, got %d", updatedRepo.KeepDays)
		}
	})

	t.Run("bad id", func(t *testing.T) {
		h := updateRepoHandler(&mock.MockRepoStore{})
		req := httptest.NewRequest(http.MethodPatch, "/repos/x", nil)
		req.SetPathValue("id", "x")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		rs := &mock.MockRepoStore{
			GetFn: func(_ context.Context, _ uint64) (*store.Repo, error) {
				return nil, store.ErrNotFound
			},
		}
		body, _ := json.Marshal(RepoUpdateRequest{})
		req := httptest.NewRequest(http.MethodPatch, "/repos/9", bytes.NewReader(body))
		req.SetPathValue("id", "9")
		rr := httptest.NewRecorder()
		h := updateRepoHandler(rs)
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("update error", func(t *testing.T) {
		rs := &mock.MockRepoStore{
			GetFn: func(_ context.Context, id uint64) (*store.Repo, error) {
				return &store.Repo{
					ID:        id,
					Name:      "x",
					Type:      store.RepoTypeFile,
					Path:      "repos/x",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
			UpdateFn: func(_ context.Context, _ *store.Repo) error {
				return errors.New("db write failed")
			},
		}
		body, _ := json.Marshal(RepoUpdateRequest{})
		req := httptest.NewRequest(http.MethodPatch, "/repos/9", bytes.NewReader(body))
		req.SetPathValue("id", "9")
		rr := httptest.NewRecorder()
		h := updateRepoHandler(rs)
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})
}

func TestCreateRepoHandlerInvalidName(t *testing.T) {
	dir := t.TempDir()
	rs := &mock.MockRepoStore{
		CreateFn: func(context.Context, *store.Repo) error {
			t.Fatal("Create must not be called when name is invalid")
			return nil
		},
	}
	body, _ := json.Marshal(RepoCreateRequest{Name: "bad name", KeepCount: 0, KeepDays: 0})
	req := httptest.NewRequest(http.MethodPut, "/repos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h := createRepoHandler(rs, dir)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid repo name, got %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("expected error message in body")
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("invalid repo name")) {
		t.Fatalf("expected body to contain 'invalid repo name', got %s", rr.Body.String())
	}
	if _, err := os.Stat(dir + "/repos"); err == nil {
		t.Fatalf("repos dir must not be created when name is invalid")
	}
}
