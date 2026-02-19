package bblt

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bak1an/artf/internal/ctxlog"
	"github.com/bak1an/artf/store"
	"go.etcd.io/bbolt"
)

type bboltAPIKeyStore struct {
	db *bbolt.DB
}

var apiIndexPrefix = []byte("apiKey")

func apiKeyIndexKey(keyHash []byte) []byte {
	return bytes.Join([][]byte{apiIndexPrefix, keyHash}, []byte(":"))
}

// Create implements [store.APIKeyStore].
func (b *bboltAPIKeyStore) Create(ctx context.Context, key *store.APIKey) error {
	log := ctxlog.From(ctx)
	log.Info("creating API key", "keyName", key.Name)

	err := b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)

		if bucket == nil {
			return fmt.Errorf("API keys bucket not found")
		}

		nextId, err := bucket.NextSequence()
		if err != nil {
			return err
		}

		log.Info("next API key ID", "id", nextId)
		key.ID = nextId

		encodedKey := u64tob(key.ID)
		var encodedValue bytes.Buffer

		encoder := gob.NewEncoder(&encodedValue)
		if err := encoder.Encode(key); err != nil {
			return err
		}

		if err := bucket.Put(encodedKey, encodedValue.Bytes()); err != nil {
			return err
		}

		log.Info("API key created", "id", key.ID)

		idxBucket := tx.Bucket(indexBucket)

		if idxBucket == nil {
			return fmt.Errorf("index bucket not found")
		}

		indexKey := apiKeyIndexKey([]byte(hex.EncodeToString(key.KeyHash)))
		if err := idxBucket.Put(indexKey, encodedKey); err != nil {
			return err
		}

		return nil
	})

	return err
}

// Delete implements [store.APIKeyStore].
func (b *bboltAPIKeyStore) Delete(ctx context.Context, id uint64) error {
	log := ctxlog.From(ctx)
	log.Info("deleting API key", "id", id)

	err := b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		if bucket == nil {
			return fmt.Errorf("API keys bucket not found")
		}

		key, err := loadAPIKey(bucket, id)
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

		indexKey := apiKeyIndexKey([]byte(hex.EncodeToString(key.KeyHash)))
		if err := idxBucket.Delete(indexKey); err != nil {
			return err
		}

		log.Info("API key deleted", "id", id)

		return nil
	})
	return err
}

// Get implements [store.APIKeyStore].
func (b *bboltAPIKeyStore) Get(ctx context.Context, id uint64) (*store.APIKey, error) {
	log := ctxlog.From(ctx)
	log.Info("getting API key", "id", id)

	var key *store.APIKey

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		if bucket == nil {
			return fmt.Errorf("API keys bucket not found")
		}
		var err error
		key, err = loadAPIKey(bucket, id)
		if err != nil {
			return err
		}
		return nil
	})

	return key, err
}

// GetByKey implements [store.APIKeyStore].
func (b *bboltAPIKeyStore) GetByKey(ctx context.Context, keyHash string) (*store.APIKey, error) {
	log := ctxlog.From(ctx)
	log.Info("getting API key by key hash", "keyHash", keyHash)

	var key *store.APIKey

	err := b.db.View(func(tx *bbolt.Tx) error {
		idxBucket := tx.Bucket(indexBucket)
		if idxBucket == nil {
			return fmt.Errorf("index bucket not found")
		}
		indexKey := apiKeyIndexKey([]byte(keyHash))
		data := idxBucket.Get(indexKey)
		if data == nil {
			return fmt.Errorf("API key not found")
		}
		if len(data) != 8 {
			return fmt.Errorf("corrupted index entry: expected 8 bytes, got %d", len(data))
		}

		bucket := tx.Bucket(apiKeysBucket)
		if bucket == nil {
			return fmt.Errorf("API keys bucket not found")
		}

		var err error
		key, err = loadAPIKey(bucket, btou64(data))
		if err != nil {
			return err
		}
		return nil
	})

	return key, err

}

// List implements [store.APIKeyStore].
func (b *bboltAPIKeyStore) List(ctx context.Context) ([]*store.APIKey, error) {
	log := ctxlog.From(ctx)
	var keys []*store.APIKey

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		if bucket == nil {
			return fmt.Errorf("API keys bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			if len(k) != 8 || v == nil {
				log.Warn("skipping unexpected entry in API keys bucket", "keyLen", len(k))
				return nil
			}
			var key store.APIKey
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&key); err != nil {
				return err
			}
			keys = append(keys, &key)
			return nil
		})
	})

	return keys, err
}

// UpdateLastUsed implements [store.APIKeyStore].
func (b *bboltAPIKeyStore) UpdateLastUsed(ctx context.Context, id uint64, lastUsed time.Time) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		if bucket == nil {
			return fmt.Errorf("API keys bucket not found")
		}

		key, err := loadAPIKey(bucket, id)
		if err != nil {
			return err
		}

		key.LastUsedAt = &lastUsed

		var encodedValue bytes.Buffer
		encoder := gob.NewEncoder(&encodedValue)
		if err := encoder.Encode(key); err != nil {
			return err
		}

		return bucket.Put(u64tob(id), encodedValue.Bytes())
	})
}

func loadAPIKey(bucket *bbolt.Bucket, id uint64) (*store.APIKey, error) {
	encodedKey := u64tob(id)
	data := bucket.Get(encodedKey)
	if data == nil {
		return nil, fmt.Errorf("API key not found")
	}

	var key store.APIKey
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&key); err != nil {
		return nil, err
	}

	return &key, nil
}
