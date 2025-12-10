# EthBackNode

<p align="center">
  <img src="assets/logo.png" alt="ethbacknode logo" width="64">
</p>

![Go](https://img.shields.io/badge/Go-1.20%2B-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-GPLv3-blue)
![JSON-RPC](https://img.shields.io/badge/API-JSON--RPC%202.0-blue)


**Backend microservice for interacting with Ethereum nodes, including transaction monitoring and transfers.**

## Overview

`ethbacknode` is a backend microservice written in **Golang** that acts as an intermediary layer between backend systems and Ethereum nodes.

Interaction with the service is performed via **JSON-RPC 2.0**, allowing seamless integration with existing backend architectures and microservice-based systems.

The service provides functionality for monitoring blockchain activity, generating Ethereum addresses, and sending transactions, including **ETH and ERC-20 token transfers**.

## Technology Stack

- **Language:** Golang
- **Service Interface:** JSON-RPC 2.0
- **Blockchain Integration:** Ethereum (EVM-compatible)
- **Ethereum Node Protocol:** Ethereum JSON-RPC

## Features

- 📡 **Transaction monitoring**
    - Track incoming and outgoing transactions
    - Monitor address activity
    - Retrieve transaction status and confirmations

- 🔐 **Ethereum address generation**
    - Generate new Ethereum addresses
    - Optional mnemonic (BIP-39) generation for created addresses
    - Backend-oriented key and mnemonic handling (implementation-dependent)

- 💸 **Transaction sending**
    - Native ETH transfers
    - ERC-20 token transfers
    - Gas and nonce management via Ethereum node RPC

- ⚙️ **Ethereum node interaction**
    - Compatible with standard Ethereum JSON-RPC nodes
    - Supports self-hosted and third-party RPC providers

## Architecture

`ethbacknode` is deployed as a standalone Go microservice:

- Exposes a **JSON-RPC 2.0 API** for client interaction
- Communicates internally with Ethereum nodes via Ethereum JSON-RPC
- Designed for backend-to-backend integration
```
[ Your Backend ]
|
JSON-RPC 2.0
|
[ ethbacknode (Go) ]
|
Ethereum JSON-RPC
|
[ Ethereum Node / RPC Provider ]
```


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

## API Interface

All client interactions with `ethbacknode` are performed via **JSON-RPC 2.0**.

The API supports operations such as:
- Monitoring transactions
- Querying address activity
- Generating Ethereum addresses and mnemonics
- Sending ETH and ERC-20 transactions

A detailed method specification will be provided in the API documentation.

## Configuration

Typical configuration parameters include:
- Ethereum RPC endpoint
- Network and chain ID
- Gas and confirmation strategy
- Key storage and signing settings

Configuration examples will be added as the project evolves.

## Security Notes

⚠️ **Important**

- Access to the JSON-RPC interface must be restricted
- Mnemonic phrases and private keys must be handled securely
- Signing endpoints should never be publicly exposed

# ethbacknode JSON-RPC 2.0 API

## API Overview

This document describes all **available API methods** and **event notifications** provided by the service.

The API is exposed via **JSON-RPC 2.0** and allows the client backend to interact with the blockchain, manage addresses, query transactions, send transfers, and receive asynchronous blockchain events.

See **API.md** for more details.

---

## Available Methods

### Service & System

- `ping` — Health check of the service
- `info` — Get blockchain and network information
- `infoGetTokenList` — Get list of supported currencies and tokens
- `infoGetBlockNum` — Get current blockchain block number

---

### Service Configuration

- `serviceConfigSet` — Configure service settings and event delivery parameters
- `serviceConfigGet` — Get current service configuration *(reserved)*

---

### Address Management

- `addressSubscribe` — Subscribe an address for blockchain notifications
- `addressGetNew` — Generate a new address and subscribe it
- `addressRecover` — Restore address data from a mnemonic *(no subscription)*
- `addressGetBalance` — Get address balances

---

### Transaction Queries

- `transferInfo` — Get detailed information about a transaction
- `transferInfoForAddress` — Get list of transactions for an address

---

### Transfers

- `transferAssets` — Send native coins or supported tokens
- `transferGetEstimatedFee` — Estimate network fee for a transfer

---

## Event Notifications

Event notifications are delivered asynchronously to the client backend via configured callback URL using **JSON-RPC 2.0**.

### Blockchain Events

- `blockEvent` — Notification about a new blockchain block

---

### Transaction Events

- `transactionEvent` — Notification about incoming or outgoing transactions  
  *(mempool, confirmation updates, and final confirmation states)*

---

## Notes

- All numeric blockchain values are provided as **big integers** unless explicitly stated
- Event delivery is **at-least-once**; clients must handle deduplication
- Critical actions and event payloads should always be verified using query methods

---

This overview serves as an entry point for the detailed sections below.

---

## General Information

- **Protocol:** JSON-RPC 2.0
- **Transport:** HTTP / HTTPS
- **Content-Type:** `application/json`

---

## Status

🚧 Project is under active development.  
Interfaces and internal behavior may change.

## License

This project is open-source and distributed under the license specified in the repository.

