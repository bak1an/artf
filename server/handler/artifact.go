package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bak1an/artf/internal/ctxlog"
	"github.com/bak1an/artf/internal/validate"
	"github.com/bak1an/artf/store"
)

type artifactInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func ListArtifacts(repos store.RepoStore, artifacts store.ArtifactStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoName := r.PathValue("repoName")

		repo, err := repos.GetByName(r.Context(), repoName)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				jsonError(w, "repo not found", http.StatusNotFound)
				return
			}
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		list, err := artifacts.ListByRepo(r.Context(), repo.ID)
		if err != nil {
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		result := make([]artifactInfo, 0, len(list))
		for _, a := range list {
			result = append(result, artifactInfo{Name: a.Name, CreatedAt: a.CreatedAt})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func DownloadArtifact(repos store.RepoStore, artifacts store.ArtifactStore, dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoName := r.PathValue("repoName")
		artifactName := r.PathValue("artifactName")

		repo, err := repos.GetByName(r.Context(), repoName)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				jsonError(w, "repo not found", http.StatusNotFound)
				return
			}
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		var artifact *store.Artifact
		if artifactName == "latest" {
			artifact, err = artifacts.GetLatestByRepo(r.Context(), repo.ID)
		} else {
			artifact, err = artifacts.GetByRepoAndName(r.Context(), repo.ID, artifactName)
		}
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				jsonError(w, "artifact not found", http.StatusNotFound)
				return
			}
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.ServeFile(w, r, filepath.Join(dataDir, artifact.Path))
	}
}

func UploadArtifact(repos store.RepoStore, artifacts store.ArtifactStore, dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())
		repoName := r.PathValue("repoName")
		artifactName := r.PathValue("artifactName")

		if err := validate.ArtifactName(artifactName); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		repo, err := repos.GetByName(r.Context(), repoName)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				jsonError(w, "repo not found", http.StatusNotFound)
				return
			}
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		repoDir := filepath.Join(dataDir, repo.Path)
		finalPath := filepath.Join(repoDir, artifactName)

		tmp, err := os.CreateTemp(repoDir, ".upload-*")
		if err != nil {
			logger.Error("failed to create temp file", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		tmpName := tmp.Name()

		if _, err := io.Copy(tmp, r.Body); err != nil {
			tmp.Close()
			os.Remove(tmpName)
			logger.Error("failed to write upload", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmpName)
			logger.Error("failed to close temp file", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		if err := os.Rename(tmpName, finalPath); err != nil {
			os.Remove(tmpName)
			logger.Error("failed to rename temp file", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		artifact := &store.Artifact{
			Name:      artifactName,
			RepoID:    repo.ID,
			Path:      repo.Path + "/" + artifactName,
			CreatedAt: time.Now().UTC(),
		}

		if err := artifacts.Create(r.Context(), artifact); err != nil {
			os.Remove(finalPath)
			logger.Error("failed to create artifact record", "error", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(artifactInfo{Name: artifact.Name, CreatedAt: artifact.CreatedAt})
	}
}
