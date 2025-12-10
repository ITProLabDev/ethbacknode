package types

type BlockInfo struct {
	BlockID      string          `json:"blockID"`
	Number       int             `json:"number"`
	ParentHash   string          `json:"parentHash"`
	Timestamp    int64           `json:"timestamp"`
	Transactions []*TransferInfo `json:"transactions"`
}
