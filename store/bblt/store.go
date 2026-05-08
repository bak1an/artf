// Package bblt is the legacy bbolt-backed implementation of store.Store.
//
// Deprecated: retained only so the one-time bbolt→sqlite migration in
// store/sqlite can read the legacy artf0.db. New code should use store/sqlite.
// Scheduled for removal once all deployments have migrated.
package bblt

import (
	"github.com/bak1an/artf/store"
	"go.etcd.io/bbolt"
)

type bboltStore struct {
	db *bbolt.DB
}

var (
	apiKeysBucket   = []byte("apiKeys")
	reposBucket     = []byte("repos")
	indexBucket     = []byte("index")
	artifactsBucket = []byte("artifacts")
)

// NewBboltStoreReadOnly wraps an already-opened read-only bbolt handle without
// touching any buckets. Intended only for the migration path in store/sqlite,
// which only calls List/Get methods on the returned store.
func NewBboltStoreReadOnly(db *bbolt.DB) store.Store {
	return &bboltStore{db: db}
}

func NewBboltStore(db *bbolt.DB) (store.Store, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(apiKeysBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(reposBucket)
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists(indexBucket)
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists(artifactsBucket)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &bboltStore{db: db}, nil
}

// Close implements [store.Store].
func (b *bboltStore) Close() error {
	return b.db.Close()
}

// Repos implements [store.Store].
func (b *bboltStore) Repos() store.RepoStore {
	return &bboltRepoStore{db: b.db}
}

// APIKeys implements [store.Store].
func (b *bboltStore) APIKeys() store.APIKeyStore {
	return &bboltAPIKeyStore{db: b.db}
}

// Artifacts implements [store.Store].
func (b *bboltStore) Artifacts() store.ArtifactStore {
	return &bboltArtifactStore{db: b.db}
}
