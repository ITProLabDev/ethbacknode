package storage

import (
	"github.com/ITProLabDev/ethbacknode/tools"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/timshannon/badgerhold"
	"os"
	"path"
	"sync"
	"time"
)

// Manager is the central storage manager that coordinates all storage backends.
// It provides thread-safe access to file storage, Badger KV, and BadgerHold databases.
type Manager struct {
	mux          sync.Mutex // Mutex for thread-safe operations
	globalDbPath string     // Base directory for all data storage
}

// NewStorageManager creates a new storage manager with the specified data directory.
// Creates the directory if it doesn't exist.
func NewStorageManager(globalDbPath string) (m *Manager, err error) {
	if !tools.IsFileExists(globalDbPath) {
		err = os.MkdirAll(globalDbPath, 0700)
		if err != nil {
			return nil, err
		}
	}
	m = &Manager{
		globalDbPath: globalDbPath,
		//simpleDatabases: make(map[string]SimpleStorage),
		//binStorages:     make(map[string]BinStorage),
	}
	return
}

// GetBinFileStorage creates and returns a file-based binary storage.
// Thread-safe operation.
func (m *Manager) GetBinFileStorage(name, moduleDbPath, moduleDbName string) (s BinStorage, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	s, err = NewBinFileStorage(name, m.globalDbPath, moduleDbPath, moduleDbName)
	if err != nil {
		return nil, err
	}
	return
}

// GetNewBadgerStorage creates and returns a new Badger key-value database.
// Thread-safe operation.
func (m *Manager) GetNewBadgerStorage(name, moduleDbPath, moduleDbName string) (s SimpleKeyStorage, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	s, err = NewBadgerStorage(name, m.globalDbPath, moduleDbPath, moduleDbName)
	if err != nil {
		return nil, err
	}
	return
}

// GetNewBadgerHoldStorage creates and returns a new BadgerHold structured database.
// Thread-safe operation.
func (m *Manager) GetNewBadgerHoldStorage(name, moduleDbPath, moduleDbName string) (s *BadgerHoldStorage, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	s, err = NewBadgerHoldStorage(name, m.globalDbPath, moduleDbPath, moduleDbName)
	if err != nil {
		return nil, err
	}
	return
}

// NewBadgerHoldStorage creates a new BadgerHold storage instance.
// BadgerHold provides ORM-like functionality on top of Badger.
func NewBadgerHoldStorage(name, globalDbPath, dbPath, dbFile string) (s *BadgerHoldStorage, err error) {
	s = new(BadgerHoldStorage)
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

// BadgerHoldStorage wraps BadgerHold for structured object storage.
// Supports auto-incrementing keys and ORM-like queries.
type BadgerHoldStorage struct {
	Name         string            `json:"-"` // Storage identifier
	GlobalDbPath string            `json:"-"` // Base data directory
	DataPath     string            `json:"-"` // Module subdirectory
	DataBaseName string            `json:"-"` // Database directory name
	db           *badgerhold.Store `json:"-"` // BadgerHold store instance
}

// connect opens the BadgerHold database, creating directories as needed.
// Retries up to 3 times on connection failure.
func (s *BadgerHoldStorage) connect() (err error) {
	dbPath := path.Join(s.GlobalDbPath, s.DataPath)
	if !s.isPathExists() {
		err = os.MkdirAll(dbPath, 0700)
		if err != nil {
			log.Error("Can not create DB dir:", err)
			return
		}
	}
	dbFile := path.Join(dbPath, s.DataBaseName)
	options := badgerhold.DefaultOptions
	options.Dir = dbFile
	options.ValueDir = dbFile
	options.Logger = nil
	s.db, err = badgerhold.Open(options)
	if err != nil {
		retryCount := 2
		for retryCount >= 0 {
			log.Warning("Can not open DB:", dbFile, "Retry:", retryCount)
			time.Sleep(1 * time.Second)
			s.db, err = badgerhold.Open(options)
			if err == nil {
				return
			}
			retryCount--
		}
	}
	return
}

// isPathExists checks if the storage directory exists.
func (s *BadgerHoldStorage) isPathExists() bool {
	if _, err := os.Stat(path.Join(s.GlobalDbPath, s.DataPath)); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// Do provides direct access to the underlying BadgerHold store.
// Use for advanced queries and operations.
func (s *BadgerHoldStorage) Do(processor func(db *badgerhold.Store)) {
	processor(s.db)
}

// Insert adds a new record with an auto-incrementing key.
func (s *BadgerHoldStorage) Insert(value interface{}) (err error) {
	return s.db.Insert(badgerhold.NextSequence(), value)
}
