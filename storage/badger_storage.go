package storage

import (
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/dgraph-io/badger"
	"os"
	"path"
	"path/filepath"
	"time"
)

// NewBadgerStorage creates a new Badger key-value database storage.
// Parameters:
//   - name: storage identifier for logging
//   - globalDbPath: base data directory
//   - dbPath: module-specific subdirectory
//   - dbFile: database directory name
func NewBadgerStorage(name, globalDbPath, dbPath, dbFile string) (s *BadgerStorage, err error) {
	s = new(BadgerStorage)
	s.Name = name
	s.GlobalDbPath = globalDbPath
	s.DataPath = dbPath
	s.DataBaseName = dbFile
	err = s.connect()
	if err != nil {
		log.Error("Can not init storage for", path.Join(dbPath, dbFile), ":", err)
	}
	return
}

// NewBadgerStorageReplicated creates a new Badger storage with replication support.
// All writes are asynchronously replicated via the provided Replicator.
func NewBadgerStorageReplicated(name, globalDbPath, dbPath, dbFile string, replicator Replicator) (s *BadgerStorageReplicated, err error) {
	s = new(BadgerStorageReplicated)
	s.Name = name
	s.GlobalDbPath = globalDbPath
	s.DataPath = dbPath
	s.DataBaseName = dbFile
	s.replicator = replicator
	err = s.connect()
	if err != nil {
		log.Error("Can not init storage for", path.Join(dbPath, dbFile), ":", err)
	}
	return
}

// BadgerStorage implements SimpleKeyStorage using Badger embedded database.
// Provides high-performance key-value storage with ACID guarantees.
type BadgerStorage struct {
	Name         string     `json:"-"` // Storage identifier
	GlobalDbPath string     `json:"-"` // Base data directory
	DataPath     string     `json:"-"` // Module subdirectory
	DataBaseName string     `json:"-"` // Database directory name
	db           *badger.DB `json:"-"` // Badger DB instance
}

// connect opens the Badger database, creating directories as needed.
// Retries up to 3 times on connection failure.
func (s *BadgerStorage) connect() (err error) {
	dbPath := path.Join(s.GlobalDbPath, s.DataPath)
	if !s.isPathExists() {
		err = os.MkdirAll(dbPath, 0700)
		if err != nil {
			log.Error("Can not create DB dir:", err)
			return
		}
	}
	dbFile := path.Join(dbPath, s.DataBaseName)
	options := badger.DefaultOptions(dbFile)
	options.Dir = dbFile
	options.ValueDir = dbFile
	options.Logger = nil
	s.db, err = badger.Open(options)
	if err != nil {
		retryCount := 2
		for retryCount >= 0 {
			log.Warning("Can not open DB:", dbFile, "Retry:", retryCount)
			time.Sleep(1 * time.Second)
			s.db, err = badger.Open(options)
			if err == nil {
				return
			}
			retryCount--
		}
	}
	return
}

// isPathExists checks if the storage directory exists.
func (s *BadgerStorage) isPathExists() bool {
	if _, err := os.Stat(filepath.Join(s.GlobalDbPath, s.DataPath)); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// Do provides direct access to the underlying Badger database.
// Use for advanced operations not covered by the interface.
func (s *BadgerStorage) Do(processor func(db *badger.DB)) {
	processor(s.db)
}

// Save persists data to the database using the data's key.
func (s *BadgerStorage) Save(value Data) (err error) {
	key := value.GetKey()
	data := value.Encode()
	return s.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, data)
		if err != nil {
			log.Error("Can not save row:", err)
		}
		return err
	})
}

// Read retrieves data by key and populates the data parameter.
// Returns badger.ErrKeyNotFound if the key doesn't exist.
func (s *BadgerStorage) Read(key Key, data Data) (err error) {
	keyBytes := key.GetKey()
	return s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(keyBytes)
		if err != nil && err != badger.ErrKeyNotFound {
			log.Error("[DB::FastStorage::Get] Can not get record:", err)
			return err
		} else if err != nil {
			return err
		}

		return item.Value(data.Decode)
	})
}

// ReadAll iterates over all records, calling processor for each value.
// Uses prefetch for efficient bulk reads.
func (s *BadgerStorage) ReadAll(processor func(raw []byte) (err error)) (err error) {
	return s.db.View(func(txn *badger.Txn) error {
		iteratorOptions := badger.DefaultIteratorOptions
		iteratorOptions.PrefetchSize = 100000
		it := txn.NewIterator(iteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(rawData []byte) error {
				return processor(rawData)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// ReadAllKey iterates over all records, calling processor with both key and value.
// Uses prefetch for efficient bulk reads.
func (s *BadgerStorage) ReadAllKey(processor func(key, raw []byte) (err error)) (err error) {
	return s.db.View(func(txn *badger.Txn) error {
		iteratorOptions := badger.DefaultIteratorOptions
		iteratorOptions.PrefetchSize = 100000
		it := txn.NewIterator(iteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(rawData []byte) error {
				key := item.Key()
				return processor(key, rawData)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete removes a record by key from the database.
func (s *BadgerStorage) Delete(rowKey []byte) (err error) {
	return s.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(rowKey)
		return err
	})
}

// Replicator defines an interface for asynchronous data replication.
type Replicator interface {
	// Update is called when data is saved.
	Update(key []byte, data []byte)
	// Delete is called when data is deleted.
	Delete(key []byte)
}

// BadgerStorageReplicated extends BadgerStorage with replication support.
// All write operations are asynchronously replicated to the configured Replicator.
type BadgerStorageReplicated struct {
	BadgerStorage
	replicator Replicator // Replication handler
}

// Save persists data and asynchronously replicates the change.
func (s *BadgerStorageReplicated) Save(value Data) (err error) {
	key := value.GetKey()
	data := value.Encode()
	go s.replicator.Update(key, data)
	return s.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, data)
		if err != nil {
			log.Error("Can not save row:", err)
		}
		return err
	})
}

// Delete removes a record and asynchronously replicates the deletion.
func (s *BadgerStorageReplicated) Delete(rowKey []byte) (err error) {
	go s.replicator.Delete(rowKey)
	return s.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(rowKey)
		return err
	})
}
