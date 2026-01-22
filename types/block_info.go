package types

// BlockInfo represents a blockchain block with its metadata and transactions.
type BlockInfo struct {
	// BlockID is the block hash (e.g., 0x...).
	BlockID string `json:"blockID"`
	// Number is the block number (height).
	Number int `json:"number"`
	// ParentHash is the hash of the parent block.
	ParentHash string `json:"parentHash"`
	// Timestamp is the Unix timestamp when the block was mined.
	Timestamp int64 `json:"timestamp"`
	// Transactions contains all transactions in this block.
	Transactions []*TransferInfo `json:"transactions"`
}
