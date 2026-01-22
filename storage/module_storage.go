package storage

import (
	"github.com/ITProLabDev/ethbacknode/tools"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"os"
	"path/filepath"
)

// ModuleManager provides module-specific storage access.
// Each module (address, watchdog, etc.) gets its own isolated storage namespace.
type ModuleManager struct {
	globalStorage     *Manager // Reference to global storage manager
	moduleName        string   // Module identifier
	moduleStoragePath string   // Module-specific subdirectory path
	init              bool     // Initialization flag
}

// GetModuleStorage creates and returns a module-specific storage manager.
// Creates the module's storage directory if it doesn't exist.
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

// GetBinFileStorage returns a file-based binary storage for the module.
// The dbName is the filename within the module's storage directory.
func (mm *ModuleManager) GetBinFileStorage(dbName string) (s BinStorage) {
	s, _ = mm.globalStorage.GetBinFileStorage(mm.moduleName, mm.moduleStoragePath, dbName)
	return
}

// GetNewBadgerStorage returns a Badger key-value storage for the module.
// The moduleDbName is the database directory name within the module's storage.
func (mm *ModuleManager) GetNewBadgerStorage(moduleDbName string) (s SimpleKeyStorage) {
	s, _ = mm.globalStorage.GetNewBadgerStorage(mm.moduleName, mm.moduleStoragePath, moduleDbName)
	return
}

// GetNewBadgerHoldStorage returns a BadgerHold structured storage for the module.
// The moduleDbName is the database directory name within the module's storage.
func (mm *ModuleManager) GetNewBadgerHoldStorage(moduleDbName string) (s *BadgerHoldStorage) {
	s, _ = mm.globalStorage.GetNewBadgerHoldStorage(mm.moduleName, mm.moduleStoragePath, moduleDbName)
	return
}
