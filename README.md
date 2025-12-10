# ethbacknode

**ethbacknode** is an open-source backend microservice for interacting with Ethereum nodes.

It provides a unified **JSON-RPC 2.0** interface for:
- tracking blockchain events and transactions
- generating and managing Ethereum addresses
- sending native ETH and ERC-20 token transfers
- delivering asynchronous blockchain events via webhooks

The service is designed for backend usage and blockchain infrastructure integration.

---

## Features

- JSON-RPC 2.0 API
- Ethereum mainnet support
- Native ETH and ERC-20 token transfers
- Address generation with mnemonic support (BIP-39)
- Transaction and balance tracking
- Asynchronous event delivery via webhooks
- Implemented in Go (Golang)
- Open-source and self-hosted

---

## Architecture Overview

ethbacknode acts as an intermediary layer between Ethereum nodes and client backend services.

All interactions with the service are performed via **JSON-RPC 2.0**.  
Blockchain events are delivered to client backends using **HTTP callbacks**.

Client Backend ⇄ JSON-RPC ⇄ ethbacknode ⇄ Ethereum Node  
⇄ Webhooks (Events)

---

## Build & Run Example

### Build from source

Clone the repository and build the binary:

git clone https://github.com/ITProLabDev/ethbacknode.git  
cd ethbacknode  
go build -o ethbacknode

### Run the service

Basic example of running the service:

./ethbacknode

The service will start and expose a JSON-RPC 2.0 endpoint for client backend interaction.

### Verify availability

You can verify that the service is running using the `ping` method:

{
"jsonrpc": "2.0",
"method": "ping",
"id": 1
}

A successful response confirms the service is operational.

---

## API Overview

ethbacknode exposes a JSON-RPC 2.0 API that includes:

- Service configuration methods
- Address management and generation
- Balance and transaction queries
- Asset transfers
- Blockchain event subscriptions

A complete and detailed API reference is available in **API.md**.

---

## Events & Webhooks

ethbacknode can notify client backends about blockchain activity, including:
- new blocks
- incoming transactions
- outgoing transactions
- transaction confirmation updates
- token transfers

Events are delivered as **JSON-RPC 2.0 POST requests** to a configured callback URL.

Event delivery is **at-least-once**, clients must handle duplicate events.

---

## Security Notes

IMPORTANT:

- Private keys and mnemonics are highly sensitive secrets
- Never expose them to frontend applications
- Never log private keys or mnemonics
- Store secrets only in secure backend storage
- Prefer watch-only mode whenever possible
- Always verify critical state using query API methods

The service does not assume responsibility for compromised credentials or lost funds.

---

## Requirements

- Go 1.20 or newer
- Access to an Ethereum node (Geth, Nethermind, Erigon, etc.)
- Network access for webhook delivery

---

## Configuration

Service behavior is configured at runtime using JSON-RPC methods.

Primary configuration method:
- serviceConfigSet

Configuration includes:
- event callback URL
- enabled event types
- token filters
- master address aggregation behavior

See **API.md** for configuration details.

---

## Open Source

ethbacknode is an open-source project intended for:
- backend engineers
- blockchain infrastructure teams
- payment and custody services
- event-driven blockchain applications

Contributions are welcome.

---

## License

GPL-3.0 license
