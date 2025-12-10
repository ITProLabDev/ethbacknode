package storage

import (
	"backnode/tools"
	"backnode/tools/log"
	"os"
	"path/filepath"
)

type ModuleManager struct {
	globalStorage     *Manager
	moduleName        string
	moduleStoragePath string
	init              bool
}

func (m *Manager) GetModuleStorage(name, dbPath string) (s *ModuleManager) {
	modukeStoragePath := filepath.Join(m.globalDbPath, dbPath)
	if !tools.IsFileExists(modukeStoragePath) {
		err := os.MkdirAll(modukeStoragePath, 0700)
		if err != nil {
			log.Error("Can not init storage for module [", name, "]:", err)
		}
	}
	s = &ModuleManager{
		globalStorage:     m,
		moduleName:        name,
		moduleStoragePath: dbPath,
	}
	return s
}

func (mm *ModuleManager) GetBinFileStorage(dbName string) (s BinStorage) {
	s, _ = mm.globalStorage.GetBinFileStorage(mm.moduleName, mm.moduleStoragePath, dbName)
	return
}

func (mm *ModuleManager) GetNewBadgerStorage(moduleDbName string) (s SimpleKeyStorage) {
	s, _ = mm.globalStorage.GetNewBadgerStorage(mm.moduleName, mm.moduleStoragePath, moduleDbName)
	return
}

func (mm *ModuleManager) GetNewBadgerHoldStorage(moduleDbName string) (s *BadgerHoldStorage) {
	s, _ = mm.globalStorage.GetNewBadgerHoldStorage(mm.moduleName, mm.moduleStoragePath, moduleDbName)
	return
}
