# EthBackNode Documentation

**Version:** 0.1.3dev
**Language:** Go 1.24
**License:** GPLv3

---

## Overview

EthBackNode is a backend microservice for interacting with Ethereum and EVM-compatible blockchains. It provides a JSON-RPC 2.0 API for address management, transaction monitoring, balance queries, and cryptocurrency transfers.

**Target Use Cases:**
- Payment processing backends
- Custodial wallet services
- Blockchain event notification systems
- DeFi application backends
- Multi-address monitoring for exchanges

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Client Applications                          │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    JSON-RPC 2.0 Endpoint (fasthttp)                 │
│                         endpoint/server.go                          │
└─────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        ▼                           ▼                           ▼
┌───────────────┐         ┌─────────────────┐         ┌─────────────────┐
│    Address    │         │  Subscriptions  │         │    TxCache      │
│    Manager    │         │    Manager      │         │    Manager      │
│  address/     │         │  subscriptions/ │         │   txcache/      │
└───────────────┘         └─────────────────┘         └─────────────────┘
        │                           │                           │
        └───────────────────────────┼───────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Watchdog Service                             │
│                         watchdog/                                   │
│              (Block/Transaction Monitoring Loop)                    │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Ethereum Chain Client                          │
│                      clients/ethclient/                             │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Universal RPC Client                             │
│                      clients/urpc/                                  │
│               (HTTP-RPC / IPC Socket Transport)                     │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Ethereum Node (geth, etc.)                       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Package Structure

### Root Package (`main`)

| File | Description |
|------|-------------|
| `main.go` | Application entry point, initialization orchestration |
| `config.go` | Global configuration management (HCL/JSON) |

### Core Packages

| Package | Path | Description |
|---------|------|-------------|
| `types` | `types/` | Core interfaces and type definitions |
| `storage` | `storage/` | Data persistence layer (Badger, files) |
| `security` | `security/` | API authentication and authorization |

### Client Packages

| Package | Path | Description |
|---------|------|-------------|
| `urpc` | `clients/urpc/` | Universal RPC client (HTTP/IPC) |
| `ethclient` | `clients/ethclient/` | Ethereum blockchain client |
| `uniclient` | `uniclient/` | Self-testing client |

### Service Packages

| Package | Path | Description |
|---------|------|-------------|
| `address` | `address/` | Address generation and pool management |
| `watchdog` | `watchdog/` | Blockchain monitoring service |
| `subscriptions` | `subscriptions/` | Event subscription management |
| `txcache` | `txcache/` | Transaction caching |
| `endpoint` | `endpoint/` | JSON-RPC HTTP server |
| `abi` | `abi/` | Smart contract ABI management |

### Cryptographic Packages

| Package | Path | Description |
|---------|------|-------------|
| `crypto` | `crypto/` | ECDSA, Keccak, transaction signing |
| `secp256k1` | `crypto/secp256k1/` | SECP256K1 curve implementation |

### Common Utilities

| Package | Path | Description |
|---------|------|-------------|
| `bip32` | `common/bip32/` | HD wallet key derivation |
| `bip39` | `common/bip39/` | Mnemonic phrase generation |
| `bip44` | `common/bip44/` | Multi-coin HD wallet support |
| `rlp` | `common/rlp/` | Recursive Length Prefix encoding |
| `base58` | `common/base58/` | Base58 encoding/decoding |
| `hexnum` | `common/hexnum/` | Hex number utilities |
| `seedphrase` | `common/seedphrase/` | Seed phrase utilities |

### Tools

| Package | Path | Description |
|---------|------|-------------|
| `log` | `tools/log/` | Structured logging |
| `file_tool` | `tools/file_tool/` | File system utilities |

---

## Configuration

### Main Configuration File (`config.hcl`)

```hcl
# Ethereum node connection
nodeUrl       = "localhost"
nodePort      = "8545"
nodeUseSSL    = false
nodeUseIPC    = true
nodeIPCSocket = "/var/tmp/geth.ipc"

# RPC endpoint server
rpcAddress = "localhost"
rpcPort    = "21280"

# Data storage
dataPath = "data"

# Debug mode
debugMode = true

# Burn address (for token tracking)
burnAddress = "0x0000000000000000000000000000000000000000"

# Optional boolean flags
flags = {
  # feature_flag = true
}

# Optional string parameters
paramsString = {
  # param_name = "value"
}

# Optional integer parameters
paramsInt = {
  confirmations = 12
}

# Additional HTTP headers for node connection
additionalHeaders = {
  X-Client = "EthBackNode/0.1.3dev"
}
```

### JSON Format (Legacy)

The configuration can also be stored in `config.json` for backward compatibility:

```json
{
  "nodeUrl": "localhost",
  "nodePort": "8545",
  "nodeUseSSL": false,
  "nodeUseIPC": true,
  "nodeIPCSocket": "/var/tmp/geth.ipc",
  "rpcAddress": "localhost",
  "rpcPort": "21280",
  "dataPath": "data",
  "debugMode": true
}
```

---

## Data Directory Structure

```
data/
├── address/
│   ├── config.json          # Address manager configuration
│   └── addresses.db/        # Badger DB for addresses
├── client/
│   └── config.json          # Chain client configuration
├── subscriptions/
│   ├── config.json          # Subscription settings
│   ├── subscribers.json     # Active subscriptions
│   └── transactions.db/     # Transaction history (BadgerHold)
├── watchdog/
│   ├── config.json          # Watchdog configuration
│   └── state.json           # Last processed block
├── txcache/
│   ├── config.json          # Cache configuration
│   └── txcache.db/          # Cached transactions (BadgerHold)
├── security/
│   └── config.json          # Security/auth configuration
└── abi/
    └── known_contracts.json # Known smart contract registry
```

---

## JSON-RPC 2.0 API

### System Methods

| Method | Description | Secured |
|--------|-------------|---------|
| `ping` | Health check | No |
| `info` | Blockchain and network info | No |
| `infoGetTokenList` | List supported tokens | No |
| `infoGetBlockNum` | Get current block number | No |

### Address Methods

| Method | Description | Secured |
|--------|-------------|---------|
| `addressSubscribe` | Subscribe address for notifications | No |
| `addressGetNew` | Generate and subscribe new address | No |
| `addressRecover` | Recover address from mnemonic | No |
| `addressGetBalance` | Query address balances | No |
| `addressGenerate` | Generate new address | Yes |

### Service Methods

| Method | Description | Secured |
|--------|-------------|---------|
| `serviceRegister` | Register service for events | No |
| `serviceConfig` | Configure service settings | Yes |

### Transaction Methods

| Method | Description | Secured |
|--------|-------------|---------|
| `transferInfo` | Get transaction details | No |
| `transferInfoForAddress` | List transactions for address | Yes |

### Transfer Methods

| Method | Description | Secured |
|--------|-------------|---------|
| `transferAssets` | Send native coins or tokens | Yes |
| `transferGetEstimatedFee` | Estimate transaction fees | Yes |

### Event Notification Methods

| Method | Description | Secured |
|--------|-------------|---------|
| `blockEvent` | New block notification | No |
| `transactionEvent` | Transaction status update | No |

---

## Core Interfaces

### ChainClient (`types/client.go`)

```go
type ChainClient interface {
    ChainClientInfo
    ChainClientMemPool
    ChainClientBlocks
    ChainClientTransactions
    ChainClientBalances
    ChainClientCoinTransfer
    ChainClientTokenTransfer
}

type ChainClientInfo interface {
    ChainId() int64
    ChainName() string
    ChainSymbol() string
    ChainDecimals() int
    ChainTokens() []TokenInfo
}

type ChainClientBalances interface {
    BalanceOf(address string) (*uint256.Int, error)
    TokensBalanceOf(address string, tokens []string) (map[string]*uint256.Int, error)
}

type ChainClientCoinTransfer interface {
    TransferByPrivateKey(privateKey *ecdsa.PrivateKey, to string, amount *uint256.Int) (*TransferInfo, error)
    TransferAllByPrivateKey(privateKey *ecdsa.PrivateKey, to string) (*TransferInfo, error)
    TransferGetEstimatedFee(from, to string, amount *uint256.Int) (*uint256.Int, error)
}

type ChainClientTokenTransfer interface {
    TransferTokenByPrivateKey(privateKey *ecdsa.PrivateKey, token, to string, amount *uint256.Int) (*TransferInfo, error)
    TransferAllTokenByPrivateKey(privateKey *ecdsa.PrivateKey, token, to string) (*TransferInfo, error)
    TransferTokenGetEstimatedFee(from, token, to string, amount *uint256.Int) (*uint256.Int, error)
}
```

### AddressCodec (`address/address.go`)

```go
type AddressCodec interface {
    EncodeBytesToAddress(bytes []byte) string
    DecodeAddressToBytes(address string) ([]byte, error)
    PrivateKeyToAddress(privateKey *ecdsa.PrivateKey) string
    IsValid(address string) bool
}
```

### Storage Interfaces (`storage/`)

```go
type BinStorage interface {
    Save(data []byte) error
    Load() ([]byte, error)
}

type SimpleStorage interface {
    Save(data Data) error
    Load(key Key) (Data, error)
    Delete(key Key) error
    ReadAll() ([]Data, error)
}
```

---

## Key Data Types

### TransferInfo (`types/transfer.go`)

```go
type TransferInfo struct {
    TxHash        string            // Transaction hash
    Timestamp     int64             // Unix timestamp
    BlockNum      int64             // Block number (-1 if pending)
    From          string            // Sender address
    To            string            // Recipient address
    Amount        *uint256.Int      // Transfer amount (wei)
    Fee           *uint256.Int      // Transaction fee (wei)
    Token         *TokenInfo        // Token info (nil for native)
    IsConfirmed   bool              // Confirmation status
    Confirmations int               // Number of confirmations
    ChainData     map[string]any    // Chain-specific data
}
```

### TokenInfo (`types/token.go`)

```go
type TokenInfo struct {
    ContractAddress string  // Token contract address
    Name            string  // Token name
    Symbol          string  // Token symbol
    Decimals        int     // Token decimals
    Protocol        string  // Protocol (e.g., "ERC-20")
}
```

### BlockInfo (`types/block.go`)

```go
type BlockInfo struct {
    BlockNum     int64           // Block number
    BlockHash    string          // Block hash
    Timestamp    int64           // Block timestamp
    Transactions []*TransferInfo // Transactions in block
}
```

### Address (`address/address.go`)

```go
type Address struct {
    Address      string   // Address string (0x...)
    AddressBytes []byte   // Address bytes (20 bytes)
    PrivateKey   []byte   // Private key bytes (32 bytes)
    IsSubscribed bool     // Subscription status
    ServiceId    string   // Subscriber service ID
    UserId       string   // User identifier
    InvoiceId    string   // Invoice identifier
    Mnemonic     string   // BIP-39 mnemonic (if generated)
    IsWatchOnly  bool     // Watch-only flag
}
```

---

## Event System

### Watchdog Events

The watchdog service emits two types of events:

```go
// Block event handler
type BlockEvent func(block *types.BlockInfo)

// Transaction event handler
type TransactionEvent func(tx *types.TransferInfo)
```

### Event Flow

1. **Watchdog** monitors blockchain for new blocks
2. For each new block, `BlockEvent` handlers are called
3. For each transaction in the block, `TransactionEvent` handlers are called
4. **Subscriptions Manager** processes events and notifies subscribers
5. **TxCache Manager** caches transaction data

### Subscriber Notification

Subscribers receive events via JSON-RPC 2.0 callbacks to their configured URLs:

```json
{
  "jsonrpc": "2.0",
  "method": "transactionEvent",
  "params": {
    "txHash": "0x...",
    "from": "0x...",
    "to": "0x...",
    "amount": "1000000000000000000",
    "blockNum": 12345678,
    "confirmations": 12
  }
}
```

---

## Cryptographic Operations

### Supported Algorithms

- **ECDSA** with SECP256K1 curve (Ethereum standard)
- **Keccak-256** hashing (Ethereum address derivation)
- **RFC 6979** deterministic ECDSA nonce generation
- **RLP** encoding for transaction serialization

### Key Generation (`crypto/ecdsa.go`)

```go
// Generate key pair from private key hex
func ECDSAKeysFromPrivateKeyHex(hexKey string) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error)

// Derive Ethereum address from public key
func PubKeyToAddressBytes(publicKey *ecdsa.PublicKey) []byte
```

### Transaction Signing (`crypto/eth_tx_signer.go`)

```go
// Sign Ethereum transaction
func SignTransaction(tx *Transaction, privateKey *ecdsa.PrivateKey) ([]byte, error)
```

---

## HD Wallet Support (BIP-32/39/44)

### Mnemonic Generation

```go
import "github.com/ITProLabDev/ethbacknode/common/bip39"

// Generate 24-word mnemonic
mnemonic, err := bip39.GenerateMnemonic(256)

// Convert mnemonic to seed
seed := bip39.MnemonicToSeed(mnemonic, "")
```

### Key Derivation

```go
import "github.com/ITProLabDev/ethbacknode/common/bip44"

// Ethereum derivation path: m/44'/60'/0'/0/index
coinType := bip44.CoinTypeETH  // 0x8000003c
```

### Address Recovery

The address manager supports recovering addresses from mnemonics using standard BIP-44 derivation paths.

---

## Storage Backends

### BinFileStorage

File-based storage for configuration files (JSON/HCL):

```go
storage := storage.NewBinFileStorage("config.json")
data, err := storage.Load()
err = storage.Save(data)
```

### BadgerStorage

High-performance key-value storage:

```go
storage := storage.NewBadgerStorage("data/addresses.db")
err := storage.Save(key, value)
value, err := storage.Load(key)
```

### BadgerHoldStorage

Structured object storage with queries:

```go
storage := storage.NewBadgerHoldStorage("data/transactions.db")
err := storage.Save(&transaction)
results, err := storage.Find(&Transaction{}, query)
```

---

## Running the Service

### Prerequisites

- Go 1.24+
- Ethereum node (geth, Nethermind, etc.)

### Build

```bash
go build -o ethbacknode .
```

### Run

```bash
# Using config file
./ethbacknode -config config.hcl

# Default (looks for config.hcl in current directory)
./ethbacknode
```

### Connect via IPC (recommended)

```hcl
nodeUseIPC    = true
nodeIPCSocket = "/path/to/geth.ipc"
```

### Connect via HTTP-RPC

```hcl
nodeUseIPC  = false
nodeUrl     = "localhost"
nodePort    = "8545"
nodeUseSSL  = false
```

---

## Testing

### Run Tests

```bash
go test ./...
```

### Test Files

| File | Package | Description |
|------|---------|-------------|
| `ecdsa_test.go` | `crypto` | ECDSA signing tests |
| `bip44_test.go` | `address` | BIP-44 derivation tests |
| `ipcclient_test.go` | `urpc` | IPC client tests |
| `client_test.go` | `uniclient` | API client tests |

---

## Dependencies

### Direct Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/hashicorp/hcl/v2` | latest | HCL configuration parsing |
| `github.com/dgraph-io/badger` | v1 | Embedded key-value database |
| `github.com/timshannon/badgerhold` | latest | ORM-like wrapper for Badger |
| `github.com/valyala/fasthttp` | latest | High-performance HTTP server |
| `github.com/holiman/uint256` | latest | 256-bit unsigned integers |
| `github.com/tyler-smith/go-bip39` | latest | BIP-39 mnemonic support |
| `golang.org/x/crypto` | latest | Cryptographic functions |

---

## Security Considerations

1. **Private Key Storage**: Private keys are stored in BadgerDB. Ensure proper filesystem permissions.

2. **API Authentication**: Secured methods require `X-Api-Token` header.

3. **IPC vs HTTP**: IPC socket connection is recommended over HTTP-RPC for security.

4. **Mnemonic Handling**: Mnemonics are stored with addresses. Consider encryption at rest.

5. **Event Callbacks**: Subscriber URLs should use HTTPS in production.

---

## Error Handling

Errors are returned as JSON-RPC 2.0 error responses:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32600,
    "message": "Invalid Request"
  }
}
```

### Standard Error Codes

| Code | Message |
|------|---------|
| -32700 | Parse error |
| -32600 | Invalid Request |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |

---

## License

This project is licensed under the GNU General Public License v3.0.
