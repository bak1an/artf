package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/bak1an/artf/internal/ctxlog"
	"github.com/bak1an/artf/store"
)

type sqliteAPIKeyStore struct {
	db *sql.DB
}

const apiKeyCols = `id, key_hash, name, read_only, created_at, last_used_at`

func scanAPIKey(s interface{ Scan(...any) error }) (*store.APIKey, error) {
	var (
		k         store.APIKey
		id        int64
		readOnly  int64
		createdAt int64
		lastUsed  sql.NullInt64
	)
	if err := s.Scan(&id, &k.KeyHash, &k.Name, &readOnly, &createdAt, &lastUsed); err != nil {
		return nil, err
	}
	k.ID = uint64(id)
	k.ReadOnly = readOnly != 0
	k.CreatedAt = time.Unix(0, createdAt).UTC()
	if lastUsed.Valid {
		t := time.Unix(0, lastUsed.Int64).UTC()
		k.LastUsedAt = &t
	}
	return &k, nil
}

// Create implements [store.APIKeyStore].
func (s *sqliteAPIKeyStore) Create(ctx context.Context, key *store.APIKey) error {
	log := ctxlog.From(ctx)
	log.Info("creating API key", "keyName", key.Name)

	var lastUsed sql.NullInt64
	if key.LastUsedAt != nil {
		lastUsed = sql.NullInt64{Int64: key.LastUsedAt.UnixNano(), Valid: true}
	}

	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO api_keys (key_hash, name, read_only, created_at, last_used_at)
		VALUES (?, ?, ?, ?, ?) RETURNING id`,
		key.KeyHash, key.Name, boolToInt(key.ReadOnly), key.CreatedAt.UnixNano(), lastUsed,
	).Scan(&id)
	if err != nil {
		return mapSQLErr(err)
	}
	key.ID = uint64(id)
	log.Info("API key created", "id", key.ID)
	return nil
}

// Get implements [store.APIKeyStore].
func (s *sqliteAPIKeyStore) Get(ctx context.Context, id uint64) (*store.APIKey, error) {
	log := ctxlog.From(ctx)
	log.Info("getting API key", "id", id)

	row := s.db.QueryRowContext(ctx, `SELECT `+apiKeyCols+` FROM api_keys WHERE id = ?`, int64(id))
	k, err := scanAPIKey(row)
	if err != nil {
		return nil, mapSQLErr(err)
	}
	return k, nil
}

// GetByKey implements [store.APIKeyStore].
func (s *sqliteAPIKeyStore) GetByKey(ctx context.Context, keyHash []byte) (*store.APIKey, error) {
	log := ctxlog.From(ctx)
	log.Debug("getting API key by key hash")

	row := s.db.QueryRowContext(ctx, `SELECT `+apiKeyCols+` FROM api_keys WHERE key_hash = ?`, keyHash)
	k, err := scanAPIKey(row)
	if err != nil {
		return nil, mapSQLErr(err)
	}
	return k, nil
}

// List implements [store.APIKeyStore].
func (s *sqliteAPIKeyStore) List(ctx context.Context) ([]*store.APIKey, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+apiKeyCols+` FROM api_keys ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*store.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// Delete implements [store.APIKeyStore].
func (s *sqliteAPIKeyStore) Delete(ctx context.Context, id uint64) error {
	log := ctxlog.From(ctx)
	log.Info("deleting API key", "id", id)

	res, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, int64(id))
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
	log.Info("API key deleted", "id", id)
	return nil
}

// UpdateLastUsed implements [store.APIKeyStore].
func (s *sqliteAPIKeyStore) UpdateLastUsed(ctx context.Context, id uint64, lastUsed time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, lastUsed.UnixNano(), int64(id))
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
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
