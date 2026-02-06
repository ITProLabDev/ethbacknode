package types

// TokenInfo represents metadata for a cryptocurrency token (e.g., ERC-20).
type TokenInfo struct {
	// ContractAddress is the smart contract address of the token.
	ContractAddress string `json:"contractAddress,omitempty"`
	// Name is the full name of the token (e.g., "Tether USD").
	Name string `json:"name"`
	// Symbol is the short symbol of the token (e.g., "USDT").
	Symbol string `json:"symbol"`
	// Decimals is the number of decimal places for the token.
	Decimals int `json:"decimals"`
	// Protocol is the token standard (e.g., "ERC-20").
	Protocol string `json:"protocol,omitempty"`
}
