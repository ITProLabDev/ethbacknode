// Package storage provides data persistence abstractions including file storage,
// key-value storage (Badger), and structured object storage (BadgerHold).
package storage

// Key interface defines a type that can provide its storage key.
type Key interface {
	// GetKey returns the byte representation of the storage key.
	GetKey() []byte
}

// Data interface defines a type that can be stored and retrieved from storage.
// It extends Key and adds encoding/decoding capabilities.
type Data interface {
	Key
	// Encode serializes the data to bytes for storage.
	Encode() []byte
	// Decode deserializes the data from bytes.
	Decode([]byte) error
}

// SimpleStorage defines a basic key-value storage interface.
type SimpleStorage interface {
	// Save persists data to storage.
	Save(data Data) (err error)
	// Read retrieves data by key and populates the data parameter.
	Read(key Key, data Data) (err error)
	// ReadAll iterates over all records, calling processor for each.
	ReadAll(processor func(raw []byte) (err error)) (err error)
	// Delete removes a record by key.
	Delete(rowKey []byte) (err error)
}

// SimpleKeyStorage extends SimpleStorage with key-aware iteration.
type SimpleKeyStorage interface {
	// Save persists data to storage.
	Save(data Data) (err error)
	// Read retrieves data by key and populates the data parameter.
	Read(key Key, data Data) (err error)
	// ReadAll iterates over all records, calling processor for each value.
	ReadAll(processor func(raw []byte) (err error)) (err error)
	// ReadAllKey iterates over all records, calling processor with both key and value.
	ReadAllKey(processor func(key, raw []byte) (err error)) (err error)
	// Delete removes a record by key.
	Delete(rowKey []byte) (err error)
}
