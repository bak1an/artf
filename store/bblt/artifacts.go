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

type bboltArtifactStore struct {
	db *bbolt.DB
}

func loadArtifact(bucket *bbolt.Bucket, id uint64) (*store.Artifact, error) {
	encodedKey := u64tob(id)
	data := bucket.Get(encodedKey)
	if data == nil {
		return nil, store.ErrNotFound
	}

	var artifact store.Artifact
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&artifact); err != nil {
		return nil, err
	}

	return &artifact, nil
}

// Create implements [store.ArtifactStore].
func (b *bboltArtifactStore) Create(ctx context.Context, artifact *store.Artifact) error {
	logger := ctxlog.From(ctx)
	logger.Info("creating artifact", "artifactName", artifact.Name, "repoID", artifact.RepoID)

	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(artifactsBucket)
		if bucket == nil {
			return fmt.Errorf("artifacts bucket not found")
		}

		nextID, err := bucket.NextSequence()
		if err != nil {
			return err
		}

		artifact.ID = nextID

		encodedKey := u64tob(artifact.ID)
		var encodedValue bytes.Buffer
		if err := gob.NewEncoder(&encodedValue).Encode(artifact); err != nil {
			return err
		}

		if err := bucket.Put(encodedKey, encodedValue.Bytes()); err != nil {
			return err
		}

		logger.Info("artifact created", "id", artifact.ID)
		return nil
	})
}

// Get implements [store.ArtifactStore].
func (b *bboltArtifactStore) Get(ctx context.Context, id uint64) (*store.Artifact, error) {
	logger := ctxlog.From(ctx)
	logger.Info("getting artifact", "id", id)

	var artifact *store.Artifact

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(artifactsBucket)
		if bucket == nil {
			return fmt.Errorf("artifacts bucket not found")
		}
		var err error
		artifact, err = loadArtifact(bucket, id)
		return err
	})

	return artifact, err
}

// List implements [store.ArtifactStore].
func (b *bboltArtifactStore) List(ctx context.Context) ([]*store.Artifact, error) {
	logger := ctxlog.From(ctx)
	var artifacts []*store.Artifact

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(artifactsBucket)
		if bucket == nil {
			return fmt.Errorf("artifacts bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			if len(k) != 8 || v == nil {
				logger.Warn("skipping unexpected entry in artifacts bucket", "keyLen", len(k))
				return nil
			}
			var artifact store.Artifact
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&artifact); err != nil {
				return err
			}
			artifacts = append(artifacts, &artifact)
			return nil
		})
	})

	return artifacts, err
}

// ListByRepo implements [store.ArtifactStore].
func (b *bboltArtifactStore) ListByRepo(ctx context.Context, repoID uint64) ([]*store.Artifact, error) {
	logger := ctxlog.From(ctx)
	logger.Info("listing artifacts by repo", "repoID", repoID)

	var artifacts []*store.Artifact

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(artifactsBucket)
		if bucket == nil {
			return fmt.Errorf("artifacts bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			if len(k) != 8 || v == nil {
				logger.Warn("skipping unexpected entry in artifacts bucket", "keyLen", len(k))
				return nil
			}
			var artifact store.Artifact
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&artifact); err != nil {
				return err
			}
			if artifact.RepoID == repoID {
				artifacts = append(artifacts, &artifact)
			}
			return nil
		})
	})

	return artifacts, err
}

// CountByRepo implements [store.ArtifactStore].
func (b *bboltArtifactStore) CountByRepo(ctx context.Context, repoID uint64) (int, error) {
	logger := ctxlog.From(ctx)
	logger.Info("counting artifacts by repo", "repoID", repoID)

	var count int

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(artifactsBucket)
		if bucket == nil {
			return fmt.Errorf("artifacts bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			if len(k) != 8 || v == nil {
				logger.Warn("skipping unexpected entry in artifacts bucket", "keyLen", len(k))
				return nil
			}
			var artifact store.Artifact
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&artifact); err != nil {
				return err
			}
			if artifact.RepoID == repoID {
				count++
			}
			return nil
		})
	})

	return count, err
}

// DeleteByRepo implements [store.ArtifactStore].
func (b *bboltArtifactStore) DeleteByRepo(ctx context.Context, repoID uint64) error {
	logger := ctxlog.From(ctx)
	logger.Info("deleting artifacts by repo", "repoID", repoID)

	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(artifactsBucket)
		if bucket == nil {
			return fmt.Errorf("artifacts bucket not found")
		}

		var keysToDelete [][]byte

		err := bucket.ForEach(func(k, v []byte) error {
			if len(k) != 8 || v == nil {
				logger.Warn("skipping unexpected entry in artifacts bucket", "keyLen", len(k))
				return nil
			}
			var artifact store.Artifact
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&artifact); err != nil {
				return err
			}
			if artifact.RepoID == repoID {
				keyCopy := make([]byte, len(k))
				copy(keyCopy, k)
				keysToDelete = append(keysToDelete, keyCopy)
			}
			return nil
		})

		if err != nil {
			return err
		}

		for _, k := range keysToDelete {
			if err := bucket.Delete(k); err != nil {
				return err
			}
		}

		logger.Info("deleted artifacts by repo", "repoID", repoID, "count", len(keysToDelete))
		return nil
	})
}

// Delete implements [store.ArtifactStore].
func (b *bboltArtifactStore) Delete(ctx context.Context, id uint64) error {
	logger := ctxlog.From(ctx)
	logger.Info("deleting artifact", "id", id)

	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(artifactsBucket)
		if bucket == nil {
			return fmt.Errorf("artifacts bucket not found")
		}

		if _, err := loadArtifact(bucket, id); err != nil {
			return err
		}

		if err := bucket.Delete(u64tob(id)); err != nil {
			return err
		}

		logger.Info("artifact deleted", "id", id)
		return nil
	})
}
