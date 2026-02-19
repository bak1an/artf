package bblt

import (
	"context"

	"github.com/bak1an/artf/store"
	"go.etcd.io/bbolt"
)

type bboltRepoStore struct {
	db *bbolt.DB
}

// Create implements [store.RepoStore].
func (b *bboltRepoStore) Create(ctx context.Context, repo *store.Repo) error {
	panic("unimplemented")
}

// Delete implements [store.RepoStore].
func (b *bboltRepoStore) Delete(ctx context.Context, id uint64) error {
	panic("unimplemented")
}

// Get implements [store.RepoStore].
func (b *bboltRepoStore) Get(ctx context.Context, id uint64) (*store.Repo, error) {
	panic("unimplemented")
}

// GetByName implements [store.RepoStore].
func (b *bboltRepoStore) GetByName(ctx context.Context, name string) (*store.Repo, error) {
	panic("unimplemented")
}

// GetByPath implements [store.RepoStore].
func (b *bboltRepoStore) GetByPath(ctx context.Context, path string) (*store.Repo, error) {
	panic("unimplemented")
}

// List implements [store.RepoStore].
func (b *bboltRepoStore) List(ctx context.Context) ([]*store.Repo, error) {
	panic("unimplemented")
}

// Update implements [store.RepoStore].
func (b *bboltRepoStore) Update(ctx context.Context, repo *store.Repo) error {
	panic("unimplemented")
}
