package filedb

import (
	"errors"
)

var (
	// IDkv is the key value for M.ID
	IDkv = KV{key: "M.ID"}

	// createdAtKV contains the key for the created at in M
	createdAtKV = KV{key: "M.CreatedAt"}

	// updatedAtKV contains the key for the updated at in M
	updatedAtKV = KV{key: "M.UpdatedAt"}

	// ErrNoDocumentFound explains itself mostly
	ErrNoDocumentFound = errors.New("Did not found requested document")
)

// KV is a key value store
type KV struct {
	key   string
	value interface{}
}

// NewKV creates a new key value store where k is the key
func NewKV(k string) KV {
	return KV{key: k}
}

// V returns the keyvalue instace with a value
func (k KV) V(v interface{}) KV {
	k.value = v
	return k
}

// dbKey returns the a part of the stored file
func (k KV) dbKey() string {
	return k.key + "=" + getHexOf(k.value) + "=" + k.key
}
