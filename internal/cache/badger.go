package cache

import (
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v3"
)

type badgerCache struct {
	db *badger.DB
}

func (bc *badgerCache) Close() error {
	return bc.db.Close()
}

func (bc *badgerCache) Set(key string, value []byte, ttl time.Duration) error {
	return bc.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

func (bc *badgerCache) SetAny(key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return bc.Set(key, b, ttl)
}

func (bc *badgerCache) Get(value string) ([]byte, error) {
	var result []byte
	err := bc.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(value))
		if err != nil {
			return err
		}

		if aErr := item.Value(func(val []byte) error {
			// This func with val would only be called if item.Value encounters no error.
			result = append([]byte{}, val...)
			return nil
		}); aErr != nil {
			return aErr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func NewBadgerCache(fl string) (Interface, error) {
	option := badger.DefaultOptions(fl)
	option.Logger = nil
	db, err := badger.Open(option)
	if err != nil {
		return nil, err
	}

	return &badgerCache{
		db: db,
	}, nil
}
