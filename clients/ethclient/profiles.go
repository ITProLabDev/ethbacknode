package ethclient

import (
	"fmt"
	"strings"

	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/types"
)

// ChainProfile describes the chain-identity defaults used to bootstrap a fresh
// client config for a target blockchain. It carries only chain-agnostic
// identity (name, label, native symbol/decimals, confirmation policy, seed
// tokens) — NOT node connection details (those live in the main config and are
// supplied by the operator). The numeric on-chain id is intentionally absent:
// it is read at runtime via eth_chainId; ChainId here is a logical label.
type ChainProfile struct {
	ChainName     string
	ChainId       string
	ChainSymbol   string
	Decimals      int
	Confirmations int
	Tokens        []*types.TokenInfo
}

// ChainProfileByName resolves a built-in chain profile by short name
// (case-insensitive). Supported: "eth" (Ethereum mainnet identity) and
// "arc" (Circle Arc testnet, whose native gas token is USDC).
func ChainProfileByName(name string) (*ChainProfile, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "eth":
		return ethProfile(), nil
	case "arc":
		return arcProfile(), nil
	default:
		return nil, fmt.Errorf("unknown chain %q (supported: Eth, Arc)", name)
	}
}

// ethProfile is the Ethereum-mainnet identity (the historical coldStart defaults).
func ethProfile() *ChainProfile {
	return &ChainProfile{
		ChainName:     "Ethereum",
		ChainId:       "ethereum",
		ChainSymbol:   "ETH",
		Decimals:      18,
		Confirmations: 20,
		Tokens: []*types.TokenInfo{
			{
				Name:            "TetherToken",
				Symbol:          "USDT",
				Decimals:        6,
				ContractAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
				Protocol:        "ERC20",
			},
			{
				Name:            "USD Coin",
				Symbol:          "USDC",
				Decimals:        6,
				ContractAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
				Protocol:        "ERC20",
			},
		},
	}
}

// arcProfile is the Circle Arc testnet identity. The native gas token is USDC;
// decimals are 18 (EVM wei). The numeric chain id (5042002) is discovered at
// runtime, not stored here. No seed tokens — addresses are added via presets.
func arcProfile() *ChainProfile {
	return &ChainProfile{
		ChainName:     "Arc Testnet",
		ChainId:       "arc-testnet",
		ChainSymbol:   "USDC",
		Decimals:      18,
		Confirmations: 1,
		Tokens:        nil,
	}
}

// InitClientConfig writes a fresh client config populated from the given chain
// profile into the supplied storage. Used by the --init bootstrap path.
func InitClientConfig(st storage.BinStorage, p *ChainProfile) error {
	if st == nil {
		return ErrConfigStorageEmpty
	}
	c := &Config{storage: st}
	c.ApplyProfile(p)
	return c.Save()
}

// ApplyProfile populates the config's chain-identity fields from a profile.
// It does not touch storage or node-connection settings.
func (c *Config) ApplyProfile(p *ChainProfile) {
	c.ChainName = p.ChainName
	c.ChainId = p.ChainId
	c.ChainSymbol = p.ChainSymbol
	c.Decimals = p.Decimals
	c.Confirmations = p.Confirmations
	c.Tokens = p.Tokens
}
