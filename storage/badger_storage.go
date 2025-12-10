package storage

import (
	"backnode/tools/log"
	"github.com/dgraph-io/badger"
	"os"
	"path"
	"path/filepath"
	"time"
)

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

type BadgerStorage struct {
	Name         string     `json:"-"`
	GlobalDbPath string     `json:"-"`
	DataPath     string     `json:"-"`
	DataBaseName string     `json:"-"`
	db           *badger.DB `json:"-"`
}

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

func (s *BadgerStorage) isPathExists() bool {
	if _, err := os.Stat(filepath.Join(s.GlobalDbPath, s.DataPath)); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (s *BadgerStorage) Do(processor func(db *badger.DB)) {
	processor(s.db)
}

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

func (s *BadgerStorage) Delete(rowKey []byte) (err error) {
	return s.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(rowKey)
		return err
	})
}

//todo create tests

type Replicator interface {
	Update(key []byte, data []byte)
	Delete(key []byte)
}

type BadgerStorageReplicated struct {
	BadgerStorage
	replicator Replicator
}

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

func (s *BadgerStorageReplicated) Delete(rowKey []byte) (err error) {
	go s.replicator.Delete(rowKey)
	return s.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(rowKey)
		return err
	})
}
