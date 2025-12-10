package watchdog

import (
	"encoding/json"
	"module github.com/ITProLabDev/ethbacknode/storage"
	"time"
)

type lastState struct {
	storage       storage.BinStorage
	setToBlock    bool
	setToBlockNum int64
	LastCheckTime time.Time `json:"lastCheckTime"`
	LastBlockNum  int64     `json:"lastBlockNum"`
}

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

func (c *lastState) UpdateState(currentBlockNum int64) error {
	c.LastCheckTime = time.Now()
	c.LastBlockNum = currentBlockNum
	return c.Save()
}

func (c *lastState) GetState() (currentBlockNum int64) {
	return c.LastBlockNum
}
