package storage

import (
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"os"
	"path"
	"path/filepath"
	"sync"
)

type BinStorage interface {
	IsExists() bool
	Save(rawData []byte) (err error)
	Load() (rawData []byte, err error)
}

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

type BinFileStorage struct {
	mux          sync.Mutex
	Name         string `json:"-"`
	GlobalDbPath string `json:"-"`
	DataPath     string `json:"-"`
	DataBaseName string `json:"-"`
}

func (s *BinFileStorage) IsExists() bool {
	return s.isFileExists()
}

func (s *BinFileStorage) filePath() string {
	if s.GlobalDbPath == "" && s.DataPath == "" {
		return s.DataBaseName
	}
	return filepath.Join(s.GlobalDbPath, s.DataPath, s.DataBaseName)
}
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

func (s *BinFileStorage) isPathExists() bool {
	path := filepath.Join(s.GlobalDbPath, s.DataPath)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (s *BinFileStorage) isFileExists() bool {
	filePath := s.filePath()
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (s *BinFileStorage) Load() (rawData []byte, err error) {
	filename := s.filePath()
	return os.ReadFile(filename)
}
