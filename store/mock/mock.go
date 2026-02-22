package mock

import (
	"context"
	"errors"
	"time"

	"github.com/bak1an/artf/store"
)

var ErrNotImplemented = errors.New("this mock method is not implemented")

// MockStore implements store.Store.
type MockStore struct {
	APIKeysFn   func() store.APIKeyStore
	ReposFn     func() store.RepoStore
	ArtifactsFn func() store.ArtifactStore
	CloseFn     func() error
}

func (m *MockStore) APIKeys() store.APIKeyStore {
	if m.APIKeysFn != nil {
		return m.APIKeysFn()
	}
	return &MockAPIKeyStore{}
}

func (m *MockStore) Repos() store.RepoStore {
	if m.ReposFn != nil {
		return m.ReposFn()
	}
	return &MockRepoStore{}
}

func (m *MockStore) Artifacts() store.ArtifactStore {
	if m.ArtifactsFn != nil {
		return m.ArtifactsFn()
	}
	return &MockArtifactStore{}
}

func (m *MockStore) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

// MockAPIKeyStore implements store.APIKeyStore.
type MockAPIKeyStore struct {
	CreateFn         func(ctx context.Context, key *store.APIKey) error
	GetFn            func(ctx context.Context, id uint64) (*store.APIKey, error)
	GetByKeyFn       func(ctx context.Context, keyHash []byte) (*store.APIKey, error)
	ListFn           func(ctx context.Context) ([]*store.APIKey, error)
	DeleteFn         func(ctx context.Context, id uint64) error
	UpdateLastUsedFn func(ctx context.Context, id uint64, lastUsed time.Time) error
}

func (m *MockAPIKeyStore) Create(ctx context.Context, key *store.APIKey) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, key)
	}
	return ErrNotImplemented
}

func (m *MockAPIKeyStore) Get(ctx context.Context, id uint64) (*store.APIKey, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, id)
	}
	return nil, ErrNotImplemented
}

func (m *MockAPIKeyStore) GetByKey(ctx context.Context, keyHash []byte) (*store.APIKey, error) {
	if m.GetByKeyFn != nil {
		return m.GetByKeyFn(ctx, keyHash)
	}
	return nil, ErrNotImplemented
}

func (m *MockAPIKeyStore) List(ctx context.Context) ([]*store.APIKey, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx)
	}
	return nil, ErrNotImplemented
}

func (m *MockAPIKeyStore) Delete(ctx context.Context, id uint64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return ErrNotImplemented
}

func (m *MockAPIKeyStore) UpdateLastUsed(ctx context.Context, id uint64, lastUsed time.Time) error {
	if m.UpdateLastUsedFn != nil {
		return m.UpdateLastUsedFn(ctx, id, lastUsed)
	}
	return ErrNotImplemented
}

// MockRepoStore implements store.RepoStore.
type MockRepoStore struct {
	CreateFn    func(ctx context.Context, repo *store.Repo) error
	GetFn       func(ctx context.Context, id uint64) (*store.Repo, error)
	GetByNameFn func(ctx context.Context, name string) (*store.Repo, error)
	ListFn      func(ctx context.Context) ([]*store.Repo, error)
	UpdateFn    func(ctx context.Context, repo *store.Repo) error
	DeleteFn    func(ctx context.Context, id uint64) error
}

func (m *MockRepoStore) Create(ctx context.Context, repo *store.Repo) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, repo)
	}
	return ErrNotImplemented
}

func (m *MockRepoStore) Get(ctx context.Context, id uint64) (*store.Repo, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, id)
	}
	return nil, ErrNotImplemented
}

func (m *MockRepoStore) GetByName(ctx context.Context, name string) (*store.Repo, error) {
	if m.GetByNameFn != nil {
		return m.GetByNameFn(ctx, name)
	}
	return nil, ErrNotImplemented
}

func (m *MockRepoStore) List(ctx context.Context) ([]*store.Repo, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx)
	}
	return nil, ErrNotImplemented
}

func (m *MockRepoStore) Update(ctx context.Context, repo *store.Repo) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, repo)
	}
	return ErrNotImplemented
}

func (m *MockRepoStore) Delete(ctx context.Context, id uint64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return ErrNotImplemented
}

// MockArtifactStore implements store.ArtifactStore.
type MockArtifactStore struct {
	CreateFn      func(ctx context.Context, artifact *store.Artifact) error
	GetFn         func(ctx context.Context, id uint64) (*store.Artifact, error)
	ListFn        func(ctx context.Context) ([]*store.Artifact, error)
	ListByRepoFn  func(ctx context.Context, repoID uint64) ([]*store.Artifact, error)
	CountByRepoFn func(ctx context.Context, repoID uint64) (int, error)
	DeleteFn      func(ctx context.Context, id uint64) error
	DeleteByRepoFn func(ctx context.Context, repoID uint64) error
}

func (m *MockArtifactStore) Create(ctx context.Context, artifact *store.Artifact) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, artifact)
	}
	return ErrNotImplemented
}

func (m *MockArtifactStore) Get(ctx context.Context, id uint64) (*store.Artifact, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, id)
	}
	return nil, ErrNotImplemented
}

func (m *MockArtifactStore) List(ctx context.Context) ([]*store.Artifact, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx)
	}
	return nil, ErrNotImplemented
}

func (m *MockArtifactStore) ListByRepo(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
	if m.ListByRepoFn != nil {
		return m.ListByRepoFn(ctx, repoID)
	}
	return nil, ErrNotImplemented
}

func (m *MockArtifactStore) CountByRepo(ctx context.Context, repoID uint64) (int, error) {
	if m.CountByRepoFn != nil {
		return m.CountByRepoFn(ctx, repoID)
	}
	return 0, ErrNotImplemented
}

func (m *MockArtifactStore) Delete(ctx context.Context, id uint64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return ErrNotImplemented
}

func (m *MockArtifactStore) DeleteByRepo(ctx context.Context, repoID uint64) error {
	if m.DeleteByRepoFn != nil {
		return m.DeleteByRepoFn(ctx, repoID)
	}
	return ErrNotImplemented
}
