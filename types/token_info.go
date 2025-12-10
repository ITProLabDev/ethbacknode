package types

type TokenInfo struct {
	ContractAddress string `json:"contractAddress,omitempty"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
	Protocol        string `json:"protocol,omitempty"`
}
