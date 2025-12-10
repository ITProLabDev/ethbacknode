# ethbacknode JSON-RPC 2.0 API

## API Overview

This document describes all **available API methods** and **event notifications** provided by the service.

The API is exposed via **JSON-RPC 2.0** and allows the client backend to interact with the blockchain, manage addresses, query transactions, send transfers, and receive asynchronous blockchain events.

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

## Methods

### ping

Health check method used to verify connectivity and service availability.

#### Parameters
None.

#### Request Example
```json
{
  "id": 0,
  "jsonrpc": "2.0",
  "method": "ping"
}
```

#### Response Example
```json
{
  "id": 0,
  "jsonrpc": "2.0",
  "result": {
    "result": "pong",
    "timestamp": 1718789894210866000
  }
}
```

#### Result Fields

| Field | Type | Description |
|-------|------|-------------|
| result | string | Always returns `pong` |
| timestamp | int64 | Server-side Unix timestamp in nanoseconds |

### info

Returns information about the connected blockchain node and supported assets.

#### Parameters
None.

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "info",
  "params": {}
}
```

#### Response Example
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "blockchain": "Ethereum",
    "id": "ethereum",
    "symbol": "ETH",
    "decimals": 18,
    "protocols": [
      "ERC20"
    ],
    "tokens": [
      {
        "contractAddress": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
        "name": "TetherToken",
        "symbol": "USDT",
        "decimals": 6,
        "protocol": "ERC20"
      }
    ]
  }
}
```

#### Result Fields

| Field | Type | Description |
|------|------|-------------|
| blockchain | string | Blockchain name |
| id | string | Internal blockchain identifier |
| symbol | string | Native network currency symbol |
| decimals | int | Native currency decimals |
| protocols | string[] | List of supported protocols |
| tokens | object[] | List of supported tokens |
| tokens[].contractAddress | string | Token smart contract address |
| tokens[].name | string | Token name |
| tokens[].symbol | string | Token symbol |
| tokens[].decimals | int | Token decimals |
| tokens[].protocol | string | Token protocol |

### infoGetTokenList

Returns the list of supported currencies and tokens available on the connected blockchain.

#### Parameters
None.

#### Request Example
```json
{
  "id": 0,
  "jsonrpc": "2.0",
  "method": "infoGetTokenList"
}
```

#### Response Example
```json
{
  "jsonrpc": "2.0",
  "id": 0,
  "result": [
    {
      "name": "Ethereum",
      "symbol": "ETH",
      "decimals": 18,
      "contractAddress": ""
    },
    {
      "name": "TetherToken",
      "symbol": "USDT",
      "decimals": 6,
      "token": true,
      "contractAddress": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380"
    }
  ]
}
```

#### Result Fields

| Field | Type | Description |
|------|------|-------------|
| result | object[] | List of supported currencies and tokens |
| result[].name | string | Currency or token name |
| result[].symbol | string | Currency or token symbol |
| result[].decimals | int | Number of decimal places |
| result[].contractAddress | string | Token smart contract address (empty for native currency) |
| result[].token | boolean | Indicates whether the item is a token |

### infoGetBlockNum

Returns the current block number of the connected blockchain.

#### Parameters
None.

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "infoGetBlockNum"
}
```

#### Response Example
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "blockNumber": 20123456
  }
}
```

#### Result Fields

| Field | Type | Description |
|------|------|-------------|
| blockNumber | int64 | Current blockchain block number |

### serviceConfigSet

Creates or updates configuration settings for a registered client service.  
The method controls webhook notifications, transaction tracking behavior, and optional fund aggregation rules.

⚠️ If a boolean parameter is **not specified**, it will be automatically set to `false` (unless explicitly stated otherwise).

#### Parameters

| Field | Type | Description |
|------|------|-------------|
| serviceId | int | Service identifier issued during service registration |
| apiToken | string | API token assigned to the service; **required if issued**, otherwise the request will be rejected |
| eventUrl | string | Client callback endpoint URL (must be valid `http://` or `https://`, may include port and URI path) |
| reportNewBlock | bool | Send notifications when new blocks are produced |
| reportIncomingTx | bool | Send notifications about incoming transactions to subscribed addresses |
| reportOutgoingTx | bool | Send notifications about outgoing transactions from subscribed addresses |
| reportMainCoin | bool | Filter notifications for the native network currency (defaults to `true` if omitted) |
| reportTokens | string[] | Filter notifications for specified token symbols |
| gatherToMaster | bool | Indicates whether received funds should be consolidated to a master address |
| masterList | string[] | List of master addresses for fund aggregation |

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "serviceConfigSet",
  "params": {
    "serviceId": 7,
    "eventUrl": "http://127.0.0.1:21100",
    "reportNewBlock": true,
    "reportIncomingTx": true,
    "reportOutgoingTx": true,
    "reportMainCoin": true,
    "reportTokens": [
      "USDT"
    ],
    "gatherToMaster": false,
    "masterList": [
      "0xfDF68CBfec145595F6943977c7Bf08d621aFd4B6"
    ]
  }
}
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "serviceId": 7,
    "eventUrl": "http://127.0.0.1:21100",
    "reportNewBlock": true,
    "reportIncomingTx": true,
    "reportOutgoingTx": true,
    "reportMainCoin": true,
    "reportTokens": [
      "USDT"
    ],
    "gatherToMaster": false,
    "masterList": [
      "0xfDF68CBfec145595F6943977c7Bf08d621aFd4B6"
    ]
  }
}
```

#### Result Fields

Response fields mirror request parameters and represent the **current service configuration state**.  
The `apiToken` field is excluded from the response.

### serviceConfigGet

**Reserved method.**

This method is reserved for future use.  
The purpose, parameters, and response format are not yet defined and may be introduced in a later API version.

At the moment, calling this method has no effect and should not be used in production integrations.

### addressSubscribe

Subscribes an address to receive blockchain event notifications.

The subscription enables notifications for incoming/outgoing transactions and other events according to the service configuration.

⚠️ **Security Warning — Private Key Usage**

Providing a `privateKey` to this method allows the service to perform sensitive operations such as automatic fund transfers or outgoing transactions.  
**Improper handling of private keys may lead to irreversible loss of funds.**

- Never transmit private keys over unsecured networks
- Use `privateKey` only for addresses dedicated to automated workflows
- Do not reuse private keys from user wallets or cold storage
- Prefer `watchOnly=true` whenever outgoing transactions are not required
- Restrict access to this method at the network and application level

The service does **not** assume responsibility for compromised keys provided by the client.

#### Parameters

| Field | Type | Description |
|------|------|-------------|
| serviceId | int | Service identifier issued during service registration |
| address | string | Ethereum address to subscribe |
| userId | string | *(optional)* Client-side user identifier; included in notifications |
| invoiceId | string | *(optional)* Client-side invoice identifier; included in notifications |
| privateKey | string | *(optional)* Private key of the address; required for automatic fund transfers or outgoing transactions |
| watchOnly | bool | *(optional)* Forces watch-only mode; disables fund transfers even if auto-gather is enabled (requires `privateKey` if false) |

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "addressSubscribe",
  "params": {
    "address": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
    "serviceId": 42,
    "watchOnly": true
  }
}
```

#### Response Example (Success)
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "success": true
  }
}
```

#### Response Example (Address Already Subscribed)
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "message": "Address already known"
  }
}
```

#### Result Fields

| Field | Type | Description |
|------|------|-------------|
| success | bool | Subscription success flag |
| message | string | Optional informational message |

### addressGetNew

Generates a new Ethereum address, automatically subscribes it for notifications, and optionally returns full address data.

If `fullInfo` is enabled, the response may include sensitive information such as the private key and mnemonic.

⚠️ **Security Warning — Private Key & Mnemonic**

When `fullInfo=true`, the service may return:
- the **private key** of the newly generated address
- the **mnemonic phrase** for address recovery

These values provide full control over the address and its funds.

- Never store or transmit private keys or mnemonics in plaintext
- Do not use generated addresses with `fullInfo=true` for cold storage
- Restrict access to this method at both network and application levels
- Prefer `watchOnly=true` if outgoing transactions are not required
- The service is not responsible for compromised keys or mnemonics

#### Parameters

| Field | Type | Description |
|------|------|-------------|
| serviceId | int | Service identifier issued during service registration |
| fullInfo | bool | *(optional)* If enabled, returns full address data including sensitive fields |
| userId | string | *(optional)* Client-side user identifier; included in notifications |
| invoiceId | string | *(optional)* Client-side invoice identifier; included in notifications |
| watchOnly | bool | *(optional)* Forces watch-only mode; disables fund transfers if enabled |

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "addressGetNew",
  "params": {
    "serviceId": 42,
    "watchOnly": true,
    "fullInfo": true
  }
}
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "address": "0x01FF05a349764C202C49e1358302fF1270d0FA77",
    "privateKey": "0x15470630d8711e77a0babff3c2416e605f3de86fb428d92f2c8ec57a8bf9e265",
    "watchOnly": true,
    "mnemonic": [
      "word1",
      "word2",
      "word3",
      "..."
    ]
  }
}
```

#### Result Fields

| Field | Type | Description |
|------|------|-------------|
| address | string | Newly generated Ethereum address (automatically subscribed) |
| privateKey | string | Private key of the address (only if `fullInfo=true`) |
| mnemonic | string[] | Mnemonic phrase for address recovery (only if `fullInfo=true`) |
| watchOnly | bool | Indicates watch-only mode |
| userId | string | Client-side user identifier (if provided) |
| invoiceId | string | Client-side invoice identifier (if provided) |

### addressRecover

Recovers Ethereum address data from a mnemonic phrase.

⚠️ **IMPORTANT — Address Recovery Only**

This method **ONLY recovers address data**.  
✅ **NO subscription is created**  
✅ **NO notifications are enabled**

The recovered address must be subscribed separately if notifications are required.

⚠️ **Security Warning — Private Key & Mnemonic**

This method returns highly sensitive data:
- the **private key** of the recovered address
- the **mnemonic phrase** (BIP-39)

Anyone with access to this information has **full and irreversible control** over the address and its funds.

- Never expose mnemonics or private keys in logs or client-side code
- Do not transmit mnemonics over unsecured channels
- Use this method only in controlled backend environments
- The service is not responsible for compromised keys or funds

#### Parameters

| Field | Type | Description |
|------|------|-------------|
| mnemonic | string[] | Mnemonic phrase (12 or 24 words) used to recover address data |

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "addressRecover",
  "params": {
    "mnemonic": [
      "fresh",
      "mosquito",
      "auction",
      "report",
      "edit",
      "cereal",
      "swing",
      "peanut",
      "brisk",
      "kick",
      "nose",
      "health"
    ]
  }
}
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "address": "0x186E9A6aF1f9F3e28D23a39478586Ac05Ca57F60",
    "privateKey": "0x1ffaeb918275b9e314c665da9b1ad54fb288ce1aa96af7ed76a3b7c45384d9a6",
    "bip39Mnemonic": [
      "fresh",
      "mosquito",
      "auction",
      "report",
      "edit",
      "cereal",
      "swing",
      "peanut",
      "brisk",
      "kick",
      "nose",
      "health"
    ]
  }
}
```

#### Result Fields

| Field | Type | Description |
|------|------|-------------|
| success | bool | Recovery operation success flag |
| address | string | Ethereum address recovered from the mnemonic |
| privateKey | string | Private key of the recovered address |
| bip39Mnemonic | string[] | Validated mnemonic phrase after post-processing |

### addressSubscribe

**Description:** Subscribe an address to receive blockchain event notifications.

**Method:** `addressSubscribe`

**Request example:**

    {
      "method": "addressSubscribe",
      "params": {
        "address": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
        "serviceId": 42,
        "watchOnly": true
      },
      "id": 1,
      "jsonrpc": "2.0"
    }

**Normal response example:**

    {
      "id": 1,
      "jsonrpc": "2.0",
      "result": {
        "success": true
      }
    }

**Response example (address already subscribed):**

    {
      "id": 1,
      "jsonrpc": "2.0",
      "result": {
        "success": true,
        "message": "Address already known"
      }
    }

**Parameters:**

- `serviceId` – integer, service ID issued during registration.
- `address` – string, address to subscribe for notifications.
- `userId` – string, optional. Arbitrary user identifier on the client side; will be included in notifications.
- `invoiceId` – string, optional. Invoice/order identifier on the client side; will be included in notifications.
- `privateKey` – string, optional. Private key for the address, required for automatic master-address sweeping or outgoing transfers initiated by the service.

  **Security warning:** Providing `privateKey` to the node backend grants full control over the funds on this address. Use only in trusted, isolated infrastructure; never expose this parameter from front-end or client devices, never log it, and avoid using test credentials in production. For maximum security prefer `watchOnly` subscriptions and external signing.

- `watchOnly` – boolean, optional. If the service configuration enables automatic sweeping to master addresses, this flag forces “watch only” mode and disables automatic transfers, even if a private key is supplied. Requires that `privateKey` is present for non-watch-only modes.

**Result fields:**

- `success` – boolean, subscription success flag.
- `message` – string, optional additional information.


---

### addressGetNew

**Description:** Generates a new address and subscribes it for notifications.

**Method:** `addressGetNew`

**Request example:**

    {
      "method": "addressGetNew",
      "params": {
        "serviceId": 42,
        "watchOnly": true,
        "fullInfo": true
      },
      "id": 1,
      "jsonrpc": "2.0"
    }

**Response example:**

    {
      "id": 1,
      "jsonrpc": "2.0",
      "result": {
        "address": "0x01FF05a349764C202C49e1358302fF1270d0FA77",
        "privateKey": "0x15470630d8711e77a0babff3c2416e605f3de86fb428d92f2c8ec57a8bf9e265",
        "watchOnly": true
      }
    }

**Parameters:**

- `serviceId` – integer, service ID issued during registration.
- `fullInfo` – boolean, optional. If `true`, the response includes full address data (see result fields).
- `userId` – string, optional. User identifier on the client side; will be included in notifications.
- `invoiceId` – string, optional. Invoice/order identifier on the client side; will be included in notifications.
- `watchOnly` – boolean, optional. If the service configuration enables automatic sweeping to master addresses, this flag forces “watch only” mode and disables automatic transfers. Requires a private key in non-watch-only scenarios.

**Result fields (when `fullInfo: true`):**

- `address` – string, newly generated address subscribed for notifications.
- `privateKey` – string, private key of the new address.

  **Security warning:** The returned `privateKey` must be handled as highly sensitive secret material. Store it only in secure key storage, do not log it, do not send it to client applications or browsers, and never commit it to version control. Anyone with this key can move all funds from the address.

- `watchOnly`, `invoiceId`, `userId` – echo of the data recorded at subscription time.
- `mnemonic` – string array, mnemonic phrase for restoring the address.


---

### addressRecover

**Description:** Restore address data from a mnemonic phrase. Only restores the address and its keys, does **not** subscribe it for notifications.

**Method:** `addressRecover`

**Important:** ONLY RESTORES ADDRESS DATA, DOES NOT CREATE A SUBSCRIPTION.

**Request example:**

    {
      "method": "addressRecover",
      "params": {
        "mnemonic": [
          "fresh",
          "mosquito",
          "auction",
          "report",
          "edit",
          "cereal",
          "swing",
          "peanut",
          "brisk",
          "kick",
          "nose",
          "health"
        ]
      },
      "id": 1,
      "jsonrpc": "2.0"
    }

**Response example:**

    {
      "id": 1,
      "jsonrpc": "2.0",
      "result": {
        "success": true,
        "address": "0x186E9A6aF1f9F3e28D23a39478586Ac05Ca57F60",
        "privateKey": "0x1ffaeb918275b9e314c665da9b1ad54fb288ce1aa96af7ed76a3b7c45384d9a6",
        "bip39Mnemonic": [
          "fresh",
          "mosquito",
          "auction",
          "report",
          "edit",
          "cereal",
          "swing",
          "peanut",
          "brisk",
          "kick",
          "nose",
          "health"
        ]
      }
    }

**Parameters:**

- `mnemonic` – string array, mnemonic phrase used to restore the address data (12/24 words).

**Result fields:**

- `success` – boolean, recovery success flag.
- `address` – string, address restored from the mnemonic.
- `privateKey` – string, private key of the restored address.
- `bip39Mnemonic` – string array, normalized BIP-39 mnemonic for verification after post-processing.

**Security warning:** Mnemonics and private keys give full control over the restored address. Keep them strictly on the backend side in secure storage, never send them to untrusted systems, and avoid persisting them in logs, analytics or error reports. Treat both `mnemonic` and `privateKey` as long-term secrets.


---

### transferInfo

**Description:** Get detailed information about a transaction.

**Method:** `transferInfo`

**Request example:**

    {
      "method": "transferInfo",
      "params": {
        "txId": "0x6389......98889cc"
      },
      "id": 1,
      "jsonrpc": "2.0"
    }

**Response example:**

    {
      "id": 1,
      "jsonrpc": "2.0",
      "result": {
        "tx_id": "0x63892............98889cc",
        "timestamp": 1719332001,
        "blockNum": 38018,
        "success": true,
        "transfer": true,
        "nativeCoin": true,
        "symbol": "ETH",
        "decimals": 18,
        "from": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
        "to": "0x8C33498C169a76dD49450fef0413e10aD9Ac98D5",
        "amount": 0.500000000000000000,
        "fee": 0.000294000000147000,
        "inPool": false,
        "confirmed": true,
        "confirmations": 50
      }
    }

**Parameters:**

- `txId` – string (hex), transaction identifier.
- `amountsFormatted` – boolean, optional, default `true`.
    - If `true`, `amount` and `fee` are returned in fixed-point decimal format.
    - If `false`, `amount` and `fee` are returned as big integers (raw units).

**Result:** See `transactionEvent` for the detailed format of `amount` and `fee` fields; `result` mirrors that structure.


---

### addressGetBalance

**Description:** Get balances for an address.

**Method:** `addressGetBalance`

**Request example:**

    {
      "method": "addressGetBalance",
      "params": {
        "address": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
        "formatted": true
      },
      "id": 1,
      "jsonrpc": "2.0"
    }

**Response example:**

    {
      "id": 1,
      "jsonrpc": "2.0",
      "result": {
        "ETH": 160000.000000000000000000,
        "USDC": 0.000000,
        "USDT": 0.000000
      }
    }

**Parameters:**

- `address` – string, address whose balance is requested.
- `formatted` – boolean, optional, default `true`.
    - If `true`, balances are returned in fixed-point decimal format.
    - If `false`, balances are returned as big integers (raw units).
- `allAssets` – boolean, optional, default `true`.
    - If `true`, returns balances for all known assets.
    - If `false`, returns only the main network currency.
- `assets` – string array, optional. List of asset symbols for which to request balances (see `info` and `infoGetTokenList`).
- `extended` – boolean, optional, reserved.

**Result:**

- `{ "symbol": balance }` – a map/associative array where key is asset symbol and value is the balance in either fixed or big-int format depending on the `formatted` parameter.

---

### transferInfo

Returns detailed information about a blockchain transfer (transaction).

This method can be used to query the current status, confirmation state, and metadata of a transaction by its hash.

#### Parameters

| Field | Type | Description |
|------|------|-------------|
| txId | string | Hex-encoded transaction identifier (transaction hash) |
| amountsFormatted | bool | *(optional, default: true)* If `true`, `amount` and `fee` are returned as fixed decimal values; if `false`, values are returned as big integers |

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "transferInfo",
  "params": {
    "txId": "0x63892............98889cc"
  }
}
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "tx_id": "0x63892............98889cc",
    "timestamp": 1719332001,
    "blockNum": 38018,
    "success": true,
    "transfer": true,
    "nativeCoin": true,
    "symbol": "ETH",
    "decimals": 18,
    "from": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
    "to": "0x8C33498C169a76dD49450fef0413e10aD9Ac98D5",
    "amount": 0.5,
    "fee": 0.000294000000147,
    "inPool": false,
    "confirmed": true,
    "confirmations": 50
  }
}
```

#### Result Fields

| Field | Type | Description |
|------|------|-------------|
| tx_id | string | Transaction hash |
| timestamp | int64 | Transaction timestamp (Unix time) |
| blockNum | int64 | Block number containing the transaction |
| success | bool | Transaction execution result |
| transfer | bool | Indicates that the transaction is a value transfer |
| nativeCoin | bool | Indicates transfer of native network currency |
| symbol | string | Currency or token symbol |
| decimals | int | Number of decimal places |
| from | string | Sender address |
| to | string | Recipient address |
| amount | number / bigint | Transfer amount |
| fee | number / bigint | Transaction fee |
| inPool | bool | Indicates whether the transaction is still in the mempool |
| confirmed | bool | Indicates whether the transaction is confirmed |
| confirmations | int | Number of confirmations |

---

### transferInfoForAddress

Returns the list of transactions associated with a specific address.

The address **must be either subscribed** via `addressSubscribe` or **previously generated** using `addressGetNew`.

#### Parameters

| Field | Type | Description |
|------|------|-------------|
| address | string | Ethereum address for which the transaction list is requested |
| amountsFormatted | bool | *(optional, default: true)* If `true`, `amount` and `fee` values are returned in fixed decimal format; if `false`, values are returned as big integers |

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "transferInfoForAddress",
  "params": {
    "address": "0x74FE1AF5DF88AC160EFEF2F1559DACEE17EDD8F3"
  }
}
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": [
    {
      "tx_id": "0x63892............98889cc",
      "timestamp": 1719332001,
      "blockNum": 38018,
      "success": true,
      "transfer": true,
      "nativeCoin": true,
      "symbol": "ETH",
      "decimals": 18,
      "from": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
      "to": "0x8C33498C169a76dD49450fef0413e10aD9Ac98D5",
      "amount": 0.5,
      "fee": 0.000294000000147,
      "inPool": false,
      "confirmed": true,
      "confirmations": 50
    }
  ]
}
```

#### Result

The result is an **array of transaction objects** in the same format as returned by `transferInfo` (transactionEvent).

Each transaction entry includes:
- transaction hash
- block and confirmation data
- sender and recipient addresses
- transfer amount and fee
- native coin or token metadata

✅ Uses the same amount formatting rules as `transferInfo`  
✅ Supports both native coin and token transfers

---

### transferAssets

Sends the native network coin or supported tokens from one address to another.

⚠️ **Important note about token transfers**  
When transferring tokens (non-native assets), the sender address MUST have enough native coin balance to pay the smart-contract execution fee.

#### Parameters

| Field | Type | Description |
|------|------|-------------|
| serviceId | int | Service identifier issued during registration |
| from | string | Sender address |
| to | string | Recipient address |
| symbol | string | Asset symbol (e.g. `ETH`, `USDT`) |
| amount | bigint | Transfer amount in smallest units (big integer) |
| privateKey | string | *(optional)* Required to sign the transaction if the address is not registered or subscribed |
| force | bool | If the address is marked as `watchOnly` and no `privateKey` is provided, forces sending funds |
| signature | any | RESERVED |

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "transferAssets",
  "params": {
    "serviceId": 42,
    "from": "0x8C33498C169a76dD49450fef0413e10aD9Ac98D5",
    "privateKey": "992...............7833",
    "symbol": "ETH",
    "to": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
    "amount": 1000000000000000000,
    "force": true
  }
}
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "tx_id": "0xf04eb4ca60c1b36400a702128bd9c98b5baa20ce7b4103bfa19688aee6276481",
    "success": true,
    "nativeCoin": true,
    "symbol": "ETH",
    "from": "0x8C33498C169a76dD49450fef0413e10aD9Ac98D5",
    "to": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
    "amount": 1000000000000000000,
    "fee": 21000000147000
  }
}
```

#### Result Fields

| Field | Type | Description |
|------|------|-------------|
| tx_id | string | Blockchain transaction hash |
| success | bool | Indicates whether the transaction was successfully sent |
| nativeCoin | bool | Indicates native network asset transfer |
| symbol | string | Asset symbol |
| from | string | Sender address |
| to | string | Recipient address |
| amount | bigint | Transferred amount |
| fee | bigint | Network transaction fee |

---

### transferGetEstimatedFee

Returns an estimated network fee for a transfer operation.

#### Parameters

| Field | Type | Description |
|------|------|-------------|
| from | string | Sender address |
| to | string | Recipient address |
| symbol | string | Asset symbol |
| amount | bigint | Transfer amount in smallest units |

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "transferGetEstimatedFee",
  "params": {
    "from": "0x8C33498C169a76dD49450fef0413e10aD9Ac98D5",
    "symbol": "ETH",
    "to": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
    "amount": 1000000000000000000
  }
}
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": 21000000147000
}
```

#### Result

The result is a **big integer** representing the estimated network fee in smallest native units.


## Events & Webhooks

This section describes **events sent by the service to the client backend** via HTTP callbacks (webhooks).

Events are generated asynchronously by the service as a result of blockchain activity or internal state changes. They are delivered to the client backend endpoint configured using `serviceConfigSet`.

### Delivery Model

- Events are sent as **HTTP POST requests**
- Payload format follows **JSON-RPC 2.0**
- The client backend must expose a publicly reachable endpoint
- The endpoint must respond with HTTP `200 OK` to confirm successful delivery

If the endpoint is unavailable or returns a non-200 status code, the service **may retry delivery** according to internal retry policies.

### Ordering and Reliability

- Events related to the same address or transaction are delivered **in chronological order**
- Delivery delays may occur due to network conditions or blockchain confirmation time
- Event delivery is **at-least-once**, clients must handle possible duplicates

### Security Considerations

- Never expose webhook endpoints publicly without proper network or application-level protection
- Validate event payloads before processing
- Do not trust event data blindly — cross-check critical information (amounts, confirmations, addresses) using API methods such as `transferInfo`
- Webhook endpoints should be isolated from public-facing services when possible

### Event Types

The service may emit events for:
- New blocks
- Incoming transactions
- Transaction confirmations
- Internal service actions related to configured addresses

Each event type is described in detail in the following sections, including payload structure and example data.

## Client Service Configuration Parameters (Events)

Event notifications are delivered to the **client backend** via the callback URL specified during service registration or configuration.

All events are sent to the configured `eventUrl` endpoint using **JSON-RPC 2.0** format.  
See `serviceConfigSet` / `serviceConfigGet` for configuration management.

The configuration parameters below define **which events are generated** and **how funds are handled** for subscribed addresses.

---

### Configuration Fields

| Field | Type | Description | Example |
|------|------|-------------|---------|
| serviceId | int | Client service identifier | `42` |
| internal | bool | Internal service flag (system use) | `false` |
| eventUrl | string | HTTP endpoint (callback URL) used to deliver events | `"http://localhost:9000/api/callback"` |
| reportNewBlock | bool | Enable notifications for new blocks | `true` |
| reportIncomingTx | bool | Enable notifications for incoming transactions to subscribed addresses | `true` |
| reportOutgoingTx | bool | Enable notifications for outgoing transactions from subscribed addresses | `true` |
| reportTokens | string array / map | List of token symbols to include in notifications | `{ "USDT": true, "USDC": true }` |
| gatherToMaster | bool | Automatically move received funds to a master address | `false` |
| masterList | string array | List of master addresses used for fund aggregation | `["0xe25226E5668C466b1a55a390DCDf91b3Bc23bFED"]` |

---

### Notes

- Events are sent **only if explicitly enabled** by configuration flags
- Token-related events are filtered by `reportTokens`
- Automatic fund aggregation (`gatherToMaster`) applies **only if `watchOnly` is disabled**
- If multiple master addresses are specified, an internal routing strategy is applied

---

### Event Delivery Format

All events sent to the client backend:
- Use **HTTP POST**
- Follow **JSON-RPC 2.0**
- Contain event-specific payloads described in the next sections

Each event type below references these configuration flags to determine whether it is emitted.

---

### blockEvent

Notification about a new block appearing in the blockchain.

This event is sent to the client backend **when a new block is detected**, provided that the `reportNewBlock` flag is enabled in the service configuration.

#### Event Format

The event is delivered to the configured `eventUrl` endpoint using **JSON-RPC 2.0** via an HTTP POST request.

#### Event Example
```json
{
  "jsonrpc": "2.0",
  "method": "blockEvent",
  "params": {
    "chainId": "ethereum",
    "blockNum": 1341,
    "blockId": "0x7f60066663da144904b1792cfd0991912342bdbbae0181b52368d72dfd5f7fe5"
  },
  "id": 1
}
```

#### Event Parameters

| Field | Type | Description |
|------|------|-------------|
| chainId | string | Blockchain identifier |
| blockNum | int | Block number |
| blockId | string (hex) | Block header hash (Block ID) |

#### Notes

- This event is informational and does not imply transaction confirmations
- Block numbering and ordering are network-specific
- Clients should treat block events as **advisory** and may cross-check block details using `infoGetBlockNum` if needed

---

### transactionEvent

Notification about a blockchain transaction.  
The format is **identical for incoming and outgoing transactions**.

This event is sent to the client backend when a transaction involving a subscribed or generated address is detected.  
Depending on configuration, events may be emitted for **mempool transactions**, **confirmed transactions**, or **both**.

#### Event Format

The event is delivered to the configured `eventUrl` endpoint using **JSON-RPC 2.0** via an HTTP POST request.

#### Event Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "transactionEvent",
  "params": {
    "chainId": "ethereum",
    "txId": "0x4b1edb1329619c67467fb916a0b78938eb878078ac59ba9afdd7a34b0646e02e",
    "timestamp": 1718803312,
    "blockNum": 0,
    "success": true,
    "transfer": true,
    "nativeCoin": true,
    "symbol": "ETH",
    "from": "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3",
    "to": "0x2a549A4d9577Eb9217E155ddc72f25866508a6A9",
    "amount": 10000000000000000000,
    "fee": 441000000000000,
    "inPool": true,
    "confirmed": false,
    "confirmations": 0
  }
}
```

#### Event Parameters

| Field | Type | Description |
|------|------|-------------|
| chainId | string | Blockchain identifier |
| txId | string (hex) | Transaction hash |
| timestamp | int | Unix timestamp when the transaction entered mempool or was included in a block |
| blockNum | int | Block number; `0` if the transaction is still in mempool |
| success | bool | Transaction execution result. **Must be checked** |
| transfer | bool | Indicates a value transfer (legacy / deprecated) |
| nativeCoin | bool | Indicates transfer of the blockchain native currency |
| symbol | string | Asset or currency symbol |
| from | string | Sender address (if applicable) |
| to | string | Recipient address |
| amount | big int | Transaction amount in smallest units |
| fee | big int | Transaction fee in smallest units |
| inPool | bool | Indicates the transaction is still in the mempool (not confirmed) |
| confirmed | bool | Indicates whether the transaction is confirmed |
| confirmations | int | Number of confirmations |
| userId | int | *(optional)* Client-side user identifier provided during address subscription |
| invoiceId | int | *(optional)* Client-side invoice identifier provided during address subscription |

#### Notes

- Transactions may be delivered **multiple times** as their state changes (e.g. mempool → confirmed)
- Clients should rely on `txId` to deduplicate events
- When `inPool = true`, the transaction is **not yet confirmed**
- Amounts and fees are provided as **big integers**; formatting to fixed decimals must be done client-side if needed
