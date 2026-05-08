package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/bak1an/artf/internal/ctxlog"
	"github.com/bak1an/artf/store"
)

type sqliteArtifactStore struct {
	db *sql.DB
}

const artifactCols = `id, name, repo_id, path, sha256, created_at`

func scanArtifact(s interface{ Scan(...any) error }) (*store.Artifact, error) {
	var (
		a         store.Artifact
		id        int64
		repoID    int64
		createdAt int64
	)
	if err := s.Scan(&id, &a.Name, &repoID, &a.Path, &a.SHA256, &createdAt); err != nil {
		return nil, err
	}
	a.ID = uint64(id)
	a.RepoID = uint64(repoID)
	a.CreatedAt = time.Unix(0, createdAt).UTC()
	return &a, nil
}

// Create implements [store.ArtifactStore].
func (s *sqliteArtifactStore) Create(ctx context.Context, artifact *store.Artifact) error {
	log := ctxlog.From(ctx)
	log.Info("creating artifact", "artifactName", artifact.Name, "repoID", artifact.RepoID)

	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO artifacts (name, repo_id, path, sha256, created_at)
		VALUES (?, ?, ?, ?, ?) RETURNING id`,
		artifact.Name, int64(artifact.RepoID), artifact.Path, artifact.SHA256, artifact.CreatedAt.UnixNano(),
	).Scan(&id)
	if err != nil {
		return mapSQLErr(err)
	}
	artifact.ID = uint64(id)
	log.Info("artifact created", "id", artifact.ID)
	return nil
}

// Get implements [store.ArtifactStore].
func (s *sqliteArtifactStore) Get(ctx context.Context, id uint64) (*store.Artifact, error) {
	log := ctxlog.From(ctx)
	log.Info("getting artifact", "id", id)

	row := s.db.QueryRowContext(ctx, `SELECT `+artifactCols+` FROM artifacts WHERE id = ?`, int64(id))
	a, err := scanArtifact(row)
	if err != nil {
		return nil, mapSQLErr(err)
	}
	return a, nil
}

// List implements [store.ArtifactStore].
func (s *sqliteArtifactStore) List(ctx context.Context) ([]*store.Artifact, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+artifactCols+` FROM artifacts ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []*store.Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, a)
	}
	return artifacts, rows.Err()
}

// ListByRepo implements [store.ArtifactStore].
func (s *sqliteArtifactStore) ListByRepo(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
	log := ctxlog.From(ctx)
	log.Info("listing artifacts by repo", "repoID", repoID)

	rows, err := s.db.QueryContext(ctx,
		`SELECT `+artifactCols+` FROM artifacts WHERE repo_id = ? ORDER BY id ASC`, int64(repoID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []*store.Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, a)
	}
	return artifacts, rows.Err()
}

// CountByRepo implements [store.ArtifactStore].
func (s *sqliteArtifactStore) CountByRepo(ctx context.Context, repoID uint64) (int, error) {
	log := ctxlog.From(ctx)
	log.Info("counting artifacts by repo", "repoID", repoID)

	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM artifacts WHERE repo_id = ?`, int64(repoID)).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// GetByRepoAndName implements [store.ArtifactStore].
func (s *sqliteArtifactStore) GetByRepoAndName(ctx context.Context, repoID uint64, name string) (*store.Artifact, error) {
	log := ctxlog.From(ctx)
	log.Info("getting artifact by repo and name", "repoID", repoID, "name", name)

	row := s.db.QueryRowContext(ctx,
		`SELECT `+artifactCols+` FROM artifacts WHERE repo_id = ? AND name = ?`, int64(repoID), name)
	a, err := scanArtifact(row)
	if err != nil {
		return nil, mapSQLErr(err)
	}
	return a, nil
}

// GetLatestByRepo implements [store.ArtifactStore].
func (s *sqliteArtifactStore) GetLatestByRepo(ctx context.Context, repoID uint64) (*store.Artifact, error) {
	log := ctxlog.From(ctx)
	log.Info("getting latest artifact by repo", "repoID", repoID)

	row := s.db.QueryRowContext(ctx,
		`SELECT `+artifactCols+` FROM artifacts WHERE repo_id = ? ORDER BY id DESC LIMIT 1`, int64(repoID))
	a, err := scanArtifact(row)
	if err != nil {
		return nil, mapSQLErr(err)
	}
	return a, nil
}

// Delete implements [store.ArtifactStore].
func (s *sqliteArtifactStore) Delete(ctx context.Context, id uint64) error {
	log := ctxlog.From(ctx)
	log.Info("deleting artifact", "id", id)

	res, err := s.db.ExecContext(ctx, `DELETE FROM artifacts WHERE id = ?`, int64(id))
	if err != nil {
		return mapSQLErr(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return store.ErrNotFound
	}
	log.Info("artifact deleted", "id", id)
	return nil
}

// DeleteByRepo implements [store.ArtifactStore].
func (s *sqliteArtifactStore) DeleteByRepo(ctx context.Context, repoID uint64) error {
	log := ctxlog.From(ctx)
	log.Info("deleting artifacts by repo", "repoID", repoID)

	res, err := s.db.ExecContext(ctx, `DELETE FROM artifacts WHERE repo_id = ?`, int64(repoID))
	if err != nil {
		return mapSQLErr(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	log.Info("deleted artifacts by repo", "repoID", repoID, "count", n)
	return nil
}
