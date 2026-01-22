package storage

import (
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"os"
	"path"
	"path/filepath"
	"sync"
)

// BinStorage defines an interface for binary file storage operations.
// Used for configuration files and other binary data.
type BinStorage interface {
	// IsExists returns true if the storage file exists.
	IsExists() bool
	// Save writes raw binary data to storage.
	Save(rawData []byte) (err error)
	// Load reads raw binary data from storage.
	Load() (rawData []byte, err error)
}

// NewBinFileStorage creates a new file-based binary storage.
// Parameters:
//   - name: storage identifier for logging
//   - globalDbPath: base data directory
//   - dbPath: module-specific subdirectory
//   - dbFile: filename
func NewBinFileStorage(name, globalDbPath, dbPath, dbFile string) (s *BinFileStorage, err error) {
	s = new(BinFileStorage)
	s.Name = name
	s.GlobalDbPath = globalDbPath
	s.DataPath = dbPath
	s.DataBaseName = dbFile
	if err != nil {
		log.Error("Can not init storage for", path.Join(globalDbPath, dbPath, dbFile), ":", err)
	}
	return
}

// BinFileStorage implements BinStorage using file system storage.
// Thread-safe for concurrent access.
type BinFileStorage struct {
	mux          sync.Mutex // Mutex for thread-safe operations
	Name         string     `json:"-"` // Storage identifier
	GlobalDbPath string     `json:"-"` // Base data directory
	DataPath     string     `json:"-"` // Module subdirectory
	DataBaseName string     `json:"-"` // Filename
}

// IsExists returns true if the storage file exists on disk.
func (s *BinFileStorage) IsExists() bool {
	return s.isFileExists()
}

// filePath returns the full path to the storage file.
func (s *BinFileStorage) filePath() string {
	if s.GlobalDbPath == "" && s.DataPath == "" {
		return s.DataBaseName
	}
	return filepath.Join(s.GlobalDbPath, s.DataPath, s.DataBaseName)
}
// Save writes raw binary data to the storage file.
// Creates the directory structure if it doesn't exist.
func (s *BinFileStorage) Save(rawData []byte) (err error) {
	filename := s.filePath()
	if !s.isPathExists() && s.DataPath != "" {
		err = os.MkdirAll(filepath.Join(s.GlobalDbPath, s.DataPath), 0700)
		if err != nil {
			return err
		}
	}
	return os.WriteFile(filename, rawData, 0644)
}

// isPathExists checks if the storage directory exists.
func (s *BinFileStorage) isPathExists() bool {
	path := filepath.Join(s.GlobalDbPath, s.DataPath)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// isFileExists checks if the storage file exists.
func (s *BinFileStorage) isFileExists() bool {
	filePath := s.filePath()
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// Load reads and returns the raw binary data from the storage file.
func (s *BinFileStorage) Load() (rawData []byte, err error) {
	filename := s.filePath()
	return os.ReadFile(filename)
}
