package watchdog

import (
	"encoding/json"
	"github.com/ITProLabDev/ethbacknode/storage"
	"time"
)

// lastState tracks the watchdog's progress through the blockchain.
// Persists the last processed block number to resume after restarts.
type lastState struct {
	storage       storage.BinStorage
	setToBlock    bool
	setToBlockNum int64
	LastCheckTime time.Time `json:"lastCheckTime"`
	LastBlockNum  int64     `json:"lastBlockNum"`
}

// Load reads the state from storage.
// Initializes to block 0 if no state exists.
func (c *lastState) Load() (err error) {
	if c.storage == nil {
		return nil
	}
	if !c.storage.IsExists() {
		err = c.UpdateState(0)
		if err != nil {
			return
		}
	}
	jsonBytes, err := c.storage.Load()
	if err != nil {
		return
	}
	err = json.Unmarshal(jsonBytes, c)
	return
}

// Save persists the state to storage as JSON.
func (c *lastState) Save() (err error) {
	if c.storage == nil {
		return nil
	}
	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return
	}
	err = c.storage.Save(data)
	return
}

// UpdateState updates and persists the last processed block number.
func (c *lastState) UpdateState(currentBlockNum int64) error {
	c.LastCheckTime = time.Now()
	c.LastBlockNum = currentBlockNum
	return c.Save()
}

// GetState returns the last processed block number.
func (c *lastState) GetState() (currentBlockNum int64) {
	return c.LastBlockNum
}
