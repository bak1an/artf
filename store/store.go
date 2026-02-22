package store

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type RepoType string

const (
	RepoTypeFile RepoType = "file" // just store files as is, nothing fancy
)

type Store interface {
	APIKeys() APIKeyStore
	Repos() RepoStore
	Close() error
}

type APIKeyStore interface {
	Create(ctx context.Context, key *APIKey) error
	Get(ctx context.Context, id uint64) (*APIKey, error)
	GetByKey(ctx context.Context, keyHash string) (*APIKey, error)
	List(ctx context.Context) ([]*APIKey, error)
	Delete(ctx context.Context, id uint64) error
	UpdateLastUsed(ctx context.Context, id uint64, lastUsed time.Time) error
}

type RepoStore interface {
	Create(ctx context.Context, repo *Repo) error
	Get(ctx context.Context, id uint64) (*Repo, error)
	GetByName(ctx context.Context, name string) (*Repo, error)
	GetByPath(ctx context.Context, path string) (*Repo, error)
	List(ctx context.Context) ([]*Repo, error)
	Update(ctx context.Context, repo *Repo) error
	Delete(ctx context.Context, id uint64) error
}

type APIKey struct {
	ID         uint64
	KeyHash    []byte
	Name       string
	ReadOnly   bool
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

type Repo struct {
	ID        uint64
	Name      string
	Type      RepoType
	Path      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
