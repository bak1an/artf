package bblt

import (
	"github.com/bak1an/artf/store"
	"go.etcd.io/bbolt"
)

type bboltStore struct {
	db *bbolt.DB
}

var (
	apiKeysBucket = []byte("apiKeys")
	reposBucket   = []byte("repos")
	indexBucket   = []byte("index")
)

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
