package bblt

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"

	"github.com/bak1an/artf/internal/ctxlog"
	"github.com/bak1an/artf/store"
	"go.etcd.io/bbolt"
)

type bboltRepoStore struct {
	db *bbolt.DB
}

var (
	repoNameIndexPrefix = []byte("repoName")
	repoPathIndexPrefix = []byte("repoPath")
)

func repoNameIndexKey(name string) []byte {
	return bytes.Join([][]byte{repoNameIndexPrefix, []byte(name)}, []byte(":"))
}

func repoPathIndexKey(path string) []byte {
	return bytes.Join([][]byte{repoPathIndexPrefix, []byte(path)}, []byte(":"))
}

func loadRepo(bucket *bbolt.Bucket, id uint64) (*store.Repo, error) {
	encodedKey := u64tob(id)
	data := bucket.Get(encodedKey)
	if data == nil {
		return nil, fmt.Errorf("repo not found")
	}

	var repo store.Repo
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// Create implements [store.RepoStore].
func (b *bboltRepoStore) Create(ctx context.Context, repo *store.Repo) error {
	log := ctxlog.From(ctx)
	log.Info("creating repo", "repoName", repo.Name)

	err := b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(reposBucket)
		if bucket == nil {
			return fmt.Errorf("repos bucket not found")
		}

		nextId, err := bucket.NextSequence()
		if err != nil {
			return err
		}

		log.Info("next repo ID", "id", nextId)
		repo.ID = nextId

		encodedKey := u64tob(repo.ID)
		var encodedValue bytes.Buffer

		encoder := gob.NewEncoder(&encodedValue)
		if err := encoder.Encode(repo); err != nil {
			return err
		}

		if err := bucket.Put(encodedKey, encodedValue.Bytes()); err != nil {
			return err
		}

		log.Info("repo created", "id", repo.ID)

		idxBucket := tx.Bucket(indexBucket)
		if idxBucket == nil {
			return fmt.Errorf("index bucket not found")
		}

		if err := idxBucket.Put(repoNameIndexKey(repo.Name), encodedKey); err != nil {
			return err
		}
		if err := idxBucket.Put(repoPathIndexKey(repo.Path), encodedKey); err != nil {
			return err
		}

		return nil
	})

	return err
}

// Delete implements [store.RepoStore].
func (b *bboltRepoStore) Delete(ctx context.Context, id uint64) error {
	log := ctxlog.From(ctx)
	log.Info("deleting repo", "id", id)

	err := b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(reposBucket)
		if bucket == nil {
			return fmt.Errorf("repos bucket not found")
		}

		repo, err := loadRepo(bucket, id)
		if err != nil {
			return err
		}

		encodedKey := u64tob(id)
		if err := bucket.Delete(encodedKey); err != nil {
			return err
		}

		idxBucket := tx.Bucket(indexBucket)
		if idxBucket == nil {
			return fmt.Errorf("index bucket not found")
		}

		if err := idxBucket.Delete(repoNameIndexKey(repo.Name)); err != nil {
			return err
		}
		if err := idxBucket.Delete(repoPathIndexKey(repo.Path)); err != nil {
			return err
		}

		log.Info("repo deleted", "id", id)

		return nil
	})
	return err
}

// Get implements [store.RepoStore].
func (b *bboltRepoStore) Get(ctx context.Context, id uint64) (*store.Repo, error) {
	log := ctxlog.From(ctx)
	log.Info("getting repo", "id", id)

	var repo *store.Repo

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(reposBucket)
		if bucket == nil {
			return fmt.Errorf("repos bucket not found")
		}
		var err error
		repo, err = loadRepo(bucket, id)
		if err != nil {
			return err
		}
		return nil
	})

	return repo, err
}

// GetByName implements [store.RepoStore].
func (b *bboltRepoStore) GetByName(ctx context.Context, name string) (*store.Repo, error) {
	log := ctxlog.From(ctx)
	log.Info("getting repo by name", "name", name)

	var repo *store.Repo

	err := b.db.View(func(tx *bbolt.Tx) error {
		idxBucket := tx.Bucket(indexBucket)
		if idxBucket == nil {
			return fmt.Errorf("index bucket not found")
		}
		indexKey := repoNameIndexKey(name)
		data := idxBucket.Get(indexKey)
		if data == nil {
			return fmt.Errorf("repo not found")
		}
		if len(data) != 8 {
			return fmt.Errorf("corrupted index entry: expected 8 bytes, got %d", len(data))
		}

		bucket := tx.Bucket(reposBucket)
		if bucket == nil {
			return fmt.Errorf("repos bucket not found")
		}

		var err error
		repo, err = loadRepo(bucket, btou64(data))
		if err != nil {
			return err
		}
		return nil
	})

	return repo, err
}

// GetByPath implements [store.RepoStore].
func (b *bboltRepoStore) GetByPath(ctx context.Context, path string) (*store.Repo, error) {
	log := ctxlog.From(ctx)
	log.Info("getting repo by path", "path", path)

	var repo *store.Repo

	err := b.db.View(func(tx *bbolt.Tx) error {
		idxBucket := tx.Bucket(indexBucket)
		if idxBucket == nil {
			return fmt.Errorf("index bucket not found")
		}
		indexKey := repoPathIndexKey(path)
		data := idxBucket.Get(indexKey)
		if data == nil {
			return fmt.Errorf("repo not found")
		}
		if len(data) != 8 {
			return fmt.Errorf("corrupted index entry: expected 8 bytes, got %d", len(data))
		}

		bucket := tx.Bucket(reposBucket)
		if bucket == nil {
			return fmt.Errorf("repos bucket not found")
		}

		var err error
		repo, err = loadRepo(bucket, btou64(data))
		if err != nil {
			return err
		}
		return nil
	})

	return repo, err
}

// List implements [store.RepoStore].
func (b *bboltRepoStore) List(ctx context.Context) ([]*store.Repo, error) {
	log := ctxlog.From(ctx)
	var repos []*store.Repo

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(reposBucket)
		if bucket == nil {
			return fmt.Errorf("repos bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			if len(k) != 8 || v == nil {
				log.Warn("skipping unexpected entry in repos bucket", "keyLen", len(k))
				return nil
			}
			var repo store.Repo
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&repo); err != nil {
				return err
			}
			repos = append(repos, &repo)
			return nil
		})
	})

	return repos, err
}

// Update implements [store.RepoStore].
func (b *bboltRepoStore) Update(ctx context.Context, repo *store.Repo) error {
	log := ctxlog.From(ctx)
	log.Info("updating repo", "id", repo.ID)

	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(reposBucket)
		if bucket == nil {
			return fmt.Errorf("repos bucket not found")
		}

		oldRepo, err := loadRepo(bucket, repo.ID)
		if err != nil {
			return err
		}

		idxBucket := tx.Bucket(indexBucket)
		if idxBucket == nil {
			return fmt.Errorf("index bucket not found")
		}

		if err := idxBucket.Delete(repoNameIndexKey(oldRepo.Name)); err != nil {
			return err
		}
		if err := idxBucket.Delete(repoPathIndexKey(oldRepo.Path)); err != nil {
			return err
		}

		var encodedValue bytes.Buffer
		encoder := gob.NewEncoder(&encodedValue)
		if err := encoder.Encode(repo); err != nil {
			return err
		}

		encodedKey := u64tob(repo.ID)
		if err := bucket.Put(encodedKey, encodedValue.Bytes()); err != nil {
			return err
		}

		if err := idxBucket.Put(repoNameIndexKey(repo.Name), encodedKey); err != nil {
			return err
		}
		if err := idxBucket.Put(repoPathIndexKey(repo.Path), encodedKey); err != nil {
			return err
		}

		log.Info("repo updated", "id", repo.ID)

		return nil
	})
}
