package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/bak1an/artf/internal/ctxlog"
	"github.com/bak1an/artf/store"
)

type sqliteRepoStore struct {
	db *sql.DB
}

const repoCols = `id, name, type, path, keep_count, keep_days, created_at, updated_at`

func scanRepo(s interface{ Scan(...any) error }) (*store.Repo, error) {
	var (
		r         store.Repo
		id        int64
		typ       string
		createdAt int64
		updatedAt int64
	)
	if err := s.Scan(&id, &r.Name, &typ, &r.Path, &r.KeepCount, &r.KeepDays, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	r.ID = uint64(id)
	r.Type = store.RepoType(typ)
	r.CreatedAt = time.Unix(0, createdAt).UTC()
	r.UpdatedAt = time.Unix(0, updatedAt).UTC()
	return &r, nil
}

// Create implements [store.RepoStore].
func (s *sqliteRepoStore) Create(ctx context.Context, repo *store.Repo) error {
	log := ctxlog.From(ctx)
	log.Info("creating repo", "repoName", repo.Name)

	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO repos (name, type, path, keep_count, keep_days, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		repo.Name, string(repo.Type), repo.Path, repo.KeepCount, repo.KeepDays,
		repo.CreatedAt.UnixNano(), repo.UpdatedAt.UnixNano(),
	).Scan(&id)
	if err != nil {
		return mapSQLErr(err)
	}
	repo.ID = uint64(id)
	log.Info("repo created", "id", repo.ID)
	return nil
}

// Get implements [store.RepoStore].
func (s *sqliteRepoStore) Get(ctx context.Context, id uint64) (*store.Repo, error) {
	log := ctxlog.From(ctx)
	log.Info("getting repo", "id", id)

	row := s.db.QueryRowContext(ctx, `SELECT `+repoCols+` FROM repos WHERE id = ?`, int64(id))
	r, err := scanRepo(row)
	if err != nil {
		return nil, mapSQLErr(err)
	}
	return r, nil
}

// GetByName implements [store.RepoStore].
func (s *sqliteRepoStore) GetByName(ctx context.Context, name string) (*store.Repo, error) {
	log := ctxlog.From(ctx)
	log.Info("getting repo by name", "name", name)

	row := s.db.QueryRowContext(ctx, `SELECT `+repoCols+` FROM repos WHERE name = ?`, name)
	r, err := scanRepo(row)
	if err != nil {
		return nil, mapSQLErr(err)
	}
	return r, nil
}

// List implements [store.RepoStore].
func (s *sqliteRepoStore) List(ctx context.Context) ([]*store.Repo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+repoCols+` FROM repos ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []*store.Repo
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

// Update implements [store.RepoStore].
func (s *sqliteRepoStore) Update(ctx context.Context, repo *store.Repo) error {
	log := ctxlog.From(ctx)
	log.Info("updating repo", "id", repo.ID)

	res, err := s.db.ExecContext(ctx, `
		UPDATE repos SET name = ?, type = ?, path = ?, keep_count = ?, keep_days = ?, updated_at = ?
		WHERE id = ?`,
		repo.Name, string(repo.Type), repo.Path, repo.KeepCount, repo.KeepDays, repo.UpdatedAt.UnixNano(), int64(repo.ID),
	)
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
	log.Info("repo updated", "id", repo.ID)
	return nil
}

// Delete implements [store.RepoStore].
func (s *sqliteRepoStore) Delete(ctx context.Context, id uint64) error {
	log := ctxlog.From(ctx)
	log.Info("deleting repo", "id", id)

	res, err := s.db.ExecContext(ctx, `DELETE FROM repos WHERE id = ?`, int64(id))
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
	log.Info("repo deleted", "id", id)
	return nil
}
