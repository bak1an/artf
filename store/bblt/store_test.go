package bblt

import (
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"
)

func TestNewBboltStoreCreatesBuckets(t *testing.T) {
	db := newTestBoltDB(t)

	_, err := NewBboltStore(db)
	if err != nil {
		t.Fatalf("NewBboltStore: %v", err)
	}

	err = db.View(func(tx *bbolt.Tx) error {
		if tx.Bucket(apiKeysBucket) == nil {
			t.Fatal("apiKeys bucket was not created")
		}
		if tx.Bucket(reposBucket) == nil {
			t.Fatal("repos bucket was not created")
		}
		if tx.Bucket(indexBucket) == nil {
			t.Fatal("index bucket was not created")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("verify buckets: %v", err)
	}
}

func TestNewBboltStoreReopenExistingDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "artf.db")

	db1, err := bbolt.Open(dbPath, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("open first db: %v", err)
	}

	st1, err := NewBboltStore(db1)
	if err != nil {
		t.Fatalf("initialize first store: %v", err)
	}

	if err := st1.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}

	db2, err := bbolt.Open(dbPath, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("open second db: %v", err)
	}

	st2, err := NewBboltStore(db2)
	if err != nil {
		t.Fatalf("initialize second store: %v", err)
	}

	if err := st2.Close(); err != nil {
		t.Fatalf("close second store: %v", err)
	}
}

func TestBboltStoreProvidesSubStores(t *testing.T) {
	_, st := newTestStore(t)

	if st.Repos() == nil {
		t.Fatal("Repos() returned nil")
	}
	if st.APIKeys() == nil {
		t.Fatal("APIKeys() returned nil")
	}
}

func TestBboltStoreClose(t *testing.T) {
	_, st := newTestStore(t)

	if err := st.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}

	// bbolt close may be idempotent; ensure repeated close does not panic.
	_ = st.Close()
}
