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

type Manager struct {
	mux          sync.Mutex
	globalDbPath string
	//simpleDatabases map[string]SimpleStorage
	//binStorages     map[string]BinStorage
}

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

func (m *Manager) GetBinFileStorage(name, moduleDbPath, moduleDbName string) (s BinStorage, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	//if s, found := m.binStorages[name]; found {
	//	return s, nil
	//}
	s, err = NewBinFileStorage(name, m.globalDbPath, moduleDbPath, moduleDbName)
	if err != nil {
		return nil, err
	}
	//m.binStorages[name] = s
	return
}

func (m *Manager) GetNewBadgerStorage(name, moduleDbPath, moduleDbName string) (s SimpleKeyStorage, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	s, err = NewBadgerStorage(name, m.globalDbPath, moduleDbPath, moduleDbName)
	if err != nil {
		return nil, err
	}
	return
}

func (m *Manager) GetNewBadgerHoldStorage(name, moduleDbPath, moduleDbName string) (s *BadgerHoldStorage, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	s, err = NewBadgerHoldStorage(name, m.globalDbPath, moduleDbPath, moduleDbName)
	if err != nil {
		return nil, err
	}
	return
}

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

type BadgerHoldStorage struct {
	Name         string            `json:"-"`
	GlobalDbPath string            `json:"-"`
	DataPath     string            `json:"-"`
	DataBaseName string            `json:"-"`
	db           *badgerhold.Store `json:"-"`
}

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

func (s *BadgerHoldStorage) isPathExists() bool {
	if _, err := os.Stat(path.Join(s.GlobalDbPath, s.DataPath)); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (s *BadgerHoldStorage) Do(processor func(db *badgerhold.Store)) {
	processor(s.db)
}

func (s *BadgerHoldStorage) Insert(value interface{}) (err error) {
	return s.db.Insert(badgerhold.NextSequence(), value)
}
