package store

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrAlreadyExists = errors.New("already exists")

type RepoType string

const (
	RepoTypeFile RepoType = "file" // just store files as is, nothing fancy
)

type Store interface {
	APIKeys() APIKeyStore
	Repos() RepoStore
	Artifacts() ArtifactStore
	Close() error
}

type APIKeyStore interface {
	Create(ctx context.Context, key *APIKey) error
	Get(ctx context.Context, id uint64) (*APIKey, error)
	GetByKey(ctx context.Context, keyHash []byte) (*APIKey, error)
	List(ctx context.Context) ([]*APIKey, error)
	Delete(ctx context.Context, id uint64) error
	UpdateLastUsed(ctx context.Context, id uint64, lastUsed time.Time) error
}

type RepoStore interface {
	Create(ctx context.Context, repo *Repo) error
	Get(ctx context.Context, id uint64) (*Repo, error)
	GetByName(ctx context.Context, name string) (*Repo, error)
	List(ctx context.Context) ([]*Repo, error)
	Update(ctx context.Context, repo *Repo) error
	Delete(ctx context.Context, id uint64) error
}

type ArtifactStore interface {
	Create(ctx context.Context, artifact *Artifact) error
	Get(ctx context.Context, id uint64) (*Artifact, error)
	List(ctx context.Context) ([]*Artifact, error)
	ListByRepo(ctx context.Context, repoID uint64) ([]*Artifact, error)
	CountByRepo(ctx context.Context, repoID uint64) (int, error)
	GetByRepoAndName(ctx context.Context, repoID uint64, name string) (*Artifact, error)
	GetLatestByRepo(ctx context.Context, repoID uint64) (*Artifact, error)
	Delete(ctx context.Context, id uint64) error
	DeleteByRepo(ctx context.Context, repoID uint64) error
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
	Path      string // path to the repo folder on the filesystem related to data directory
	KeepCount int    // how many artifacts to keep, 0 means keep all
	KeepDays  int    // how many days to keep artifacts, 0 means keep all
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Artifact struct {
	ID        uint64
	Name      string
	RepoID    uint64
	Path      string // path to the artifact folder on the filesystem related to data directory
	CreatedAt time.Time
}
