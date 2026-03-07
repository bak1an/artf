package admin

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bak1an/artf/internal/ctxlog"
	"github.com/bak1an/artf/store"
)

func listReposHandler(repoStore store.RepoStore, artifactStore store.ArtifactStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())

		repos, err := repoStore.List(r.Context())
		if err != nil {
			logger.Error("failed to list repos", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := RepoListResponse{
			Repos: make([]*Repo, len(repos)),
		}
		for i, repo := range repos {
			count, err := artifactStore.CountByRepo(r.Context(), repo.ID)
			if err != nil {
				logger.Error("failed to count artifacts by repo", "error", err, "repoID", repo.ID)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			resp.Repos[i] = &Repo{
				ID:            repo.ID,
				Name:          repo.Name,
				KeepCount:     repo.KeepCount,
				KeepDays:      repo.KeepDays,
				CreatedAt:     repo.CreatedAt,
				ArtifactCount: count,
			}
		}

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.Error("failed to encode response", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func createRepoHandler(repoStore store.RepoStore, dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())

		var req RepoCreateRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Error("failed to decode request", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		now := time.Now()
		repo := &store.Repo{
			Name:      req.Name,
			Type:      store.RepoTypeFile, // Only one type for now
			Path:      "repos/" + req.Name,
			KeepCount: req.KeepCount,
			KeepDays:  req.KeepDays,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err = repoStore.Create(r.Context(), repo)
		if err != nil {
			logger.Error("failed to create repo", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		repoDir := filepath.Join(dataDir, repo.Path)
		if err := os.MkdirAll(repoDir, 0770); err != nil {
			logger.Error("failed to create repo directory", "error", err, "dir", repoDir)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(Repo{
			ID:        repo.ID,
			Name:      repo.Name,
			KeepCount: repo.KeepCount,
			KeepDays:  repo.KeepDays,
			CreatedAt: repo.CreatedAt,
		})
		if err != nil {
			logger.Error("failed to encode response", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func getRepoInfoHandler(repoStore store.RepoStore, artifactStore store.ArtifactStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())
		idPath := r.PathValue("id")
		id, err := strconv.ParseUint(idPath, 10, 64)
		if err != nil {
			logger.Error("failed to parse repo ID", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		repo, err := repoStore.Get(r.Context(), id)
		if err == store.ErrNotFound {
			logger.Error("repo not found", "id", id)
			http.Error(w, "repo not found", http.StatusNotFound)
			return
		} else if err != nil {
			logger.Error("failed to get repo", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		artifacts, err := artifactStore.ListByRepo(r.Context(), id)
		if err != nil {
			logger.Error("failed to list artifacts by repo", "error", err, "repoID", id)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := RepoInfoResponse{
			Repo: RepoInfo{
				ID:            repo.ID,
				Name:          repo.Name,
				Type:          string(repo.Type),
				Path:          repo.Path,
				KeepCount:     repo.KeepCount,
				KeepDays:      repo.KeepDays,
				CreatedAt:     repo.CreatedAt,
				UpdatedAt:     repo.UpdatedAt,
				ArtifactCount: len(artifacts),
			},
			Artifacts: make([]*ArtifactInfo, len(artifacts)),
		}

		for i, artifact := range artifacts {
			resp.Artifacts[i] = &ArtifactInfo{
				ID:        artifact.ID,
				Name:      artifact.Name,
				RepoID:    artifact.RepoID,
				Path:      artifact.Path,
				CreatedAt: artifact.CreatedAt,
			}
		}

		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.Error("failed to encode response", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func deleteRepoHandler(repoStore store.RepoStore, artifactStore store.ArtifactStore, dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())
		idPath := r.PathValue("id")
		id, err := strconv.ParseUint(idPath, 10, 64)
		if err != nil {
			logger.Error("failed to parse repo ID", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		repo, err := repoStore.Get(r.Context(), id)
		if err == store.ErrNotFound {
			logger.Error("repo not found", "id", id)
			http.Error(w, "repo not found", http.StatusNotFound)
			return
		} else if err != nil {
			logger.Error("failed to get repo", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := artifactStore.DeleteByRepo(r.Context(), id); err != nil {
			logger.Error("failed to delete artifacts by repo", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := repoStore.Delete(r.Context(), id); err != nil {
			logger.Error("failed to delete repo", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		repoDir := filepath.Join(dataDir, repo.Path)
		if err := os.RemoveAll(repoDir); err != nil {
			logger.Error("failed to remove repo directory", "error", err, "dir", repoDir)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
