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

### Smart Contracts (Universal Contract Layer)

Describe **any** smart contract by its standard JSON ABI, then receive its events
**and** the transactions sent to it, and call its read-only view methods. Both
`dot.case` and `camelCase` names work.

- `contractRegister` — Register a contract + its ABI *(secured)*
- `contractSubscribe` — Subscribe a service to a contract's events **and**
  transactions with a `scope` *(secured)*
- `contractUnsubscribe` — Remove a subscription *(secured)*
- `contractList` — List registered contracts (name → address)
- `contractSubscriptions` — List active subscriptions
- `contractCall` — Call a read-only view method, returns decoded outputs

One `contractSubscribe` call yields **both** notification types — there is no
separate subscribe step for transactions. See the
**[Smart Contract Layer](#smart-contract-layer)** section for full details.

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

### Contract Events

- `contractEvent` — A decoded smart-contract log event, delivered to a subscriber
  whose `scope` matched. Carries the event name, contract address, block/tx
  context, and the fully decoded parameters. See
  **[contractEvent](#contractevent)**.

---

### Contract Transactions

- `contractTransaction` — A transaction sent directly to a subscribed contract,
  delivered whether or not it emitted any event (including reverted calls).
  Carries the transaction context, its outcome, and its call decoded by method
  selector — plus the raw calldata either way. Delivered from the **same**
  subscription as `contractEvent`, `whole_contract` scope only for now. See
  **[contractTransaction](#contracttransaction)**.

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

---

# Smart Contract Layer

The **Universal Contract Layer** lets you work with **arbitrary** smart contracts —
not just the built-in native coin and ERC-20 tokens. You describe a contract by
its standard Ethereum JSON ABI (the same ABI you get from Etherscan/Polygonscan
or `solc`), and the service can then:

1. **Decode and deliver its events** to your backend as `contractEvent`
   notifications (push, over your webhook).
2. **Deliver every transaction sent to it** — decoded, whatever its outcome —
   as `contractTransaction` notifications (push, same webhook, same
   subscription as 1).
3. **Call its read-only view methods** on demand via `contractCall` (pull).
4. **List** what is registered and what you are subscribed to.

> This is a **generic** engine. "Polymarket-class" (multi-token ERC-1155,
> tuple/struct orders, rich events) only denotes the *complexity* it can handle;
> there is no contract-specific logic baked in.

### Quick start (the happy path)

```
1. contractRegister   — register the contract + its ABI            (once)
2. serviceConfigSet    — make sure your eventUrl webhook is set     (once)
3. contractSubscribe   — subscribe your serviceId with a scope      (once)
   → from now on, matching events arrive at your webhook as `contractEvent`,
     and (whole_contract scope) every transaction to the contract arrives as
     `contractTransaction` — one subscribe step, both notification types
4. contractCall        — read view methods (totalSupply, …) anytime (on demand)
```

**Ordering matters:** you must `contractRegister` a contract **before** you
`contractSubscribe` to it. Subscribing to an address with no registered ABI is
**rejected** (`unknown contract: register its ABI before subscribing`) — events
of an unknown contract cannot be decoded, so the subscription would deliver
nothing. Register first, then subscribe.

### Method summary

| Method (dot.case / camelCase) | Secured | Purpose |
|-------------------------------|:-------:|---------|
| `contract.register` / `contractRegister` | 🔒 | Register a contract + canonical JSON ABI |
| `contract.subscribe` / `contractSubscribe` | 🔒 | Subscribe a `serviceId` to a contract's events **and** transactions with a `scope` |
| `contract.unsubscribe` / `contractUnsubscribe` | 🔒 | Remove a subscription |
| `contract.list` / `contractList` | open | List registered contracts (name → address) |
| `contract.subscriptions` / `contractSubscriptions` | open | List active subscriptions |
| `contract.call` / `contractCall` | open | Read-only view-method call → decoded outputs |

🔒 **Secured** methods require a valid `serviceId` **and** the matching API token
(`X-Api-Token` header) for that service, exactly like `serviceConfigSet` and the
transfer methods. The subscriber service must already exist (register it via
`serviceRegister` first) — you subscribe an *existing* service to contract events.

### Event scopes

When you `contractSubscribe`, you choose a **scope** that decides *which* of the
contract's events are delivered to you:

| Scope | Delivers |
|-------|----------|
| `whole_contract` | **Every** event emitted by the contract, regardless of who is involved. |
| `managed_only` | **Only** events that involve a **managed address** — an address known to this node's address pool (subscribed/generated here). A match is made when any `address`-typed parameter of the decoded event equals a managed address. |

`managed_only` is matched on the **decoded** event parameters using the same
EIP-55 checksummed address encoding the node uses internally, so it matches
reliably regardless of how the address casing appears on-chain.

The two scopes can coexist: different services (or the same service on different
contracts) may use different scopes. Scope is chosen per subscription at
subscribe time, not globally.

> **Scope and `contractTransaction`:** the table above describes `contractEvent`.
> `contractTransaction` currently honors only `whole_contract` — a `managed_only`
> subscription still receives `contractEvent` as normal but **not**
> `contractTransaction`. Extending `managed_only` to transactions (matched on
> `from`/`to` instead of decoded event parameters) is planned but not built yet.
> A `whole_contract` subscription can still narrow `contractTransaction` to
> specific operations — see
> [Filtering contractTransaction by operation](#filtering-contracttransaction-by-operation).

---

## contractRegister

🔒 **Secured.** Registers a smart contract and its ABI so the service can decode
its events and call its methods. Registration is **persistent** — the contract
survives restarts. Registering the same address again updates nothing new (it is
deduplicated by address).

**Method:** `contractRegister` (alias `contract.register`)

#### Parameters

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| serviceId | int | ✅ | Your service identifier (also used for auth) |
| address | string | ✅ | The contract address (`0x…`, EIP-55 or lowercase both accepted) |
| abi | string | ✅ | The contract's ABI as a **JSON string** (canonical Etherscan/`solc` array form, or the project's `{ "entries": [...] }` form) |
| name | string | optional | A human-readable name for the contract (used in `contractList`) |
| symbol | string | optional | A short symbol/tag for the contract |

> **`abi` is a JSON *string***, i.e. the ABI document serialized into a single
> string value — not a nested JSON object. See the example.

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "contractRegister",
  "params": {
    "serviceId": 42,
    "name": "MultiToken",
    "symbol": "MT",
    "address": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "abi": "[{\"type\":\"event\",\"name\":\"TransferSingle\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"indexed\":true},{\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true},{\"name\":\"id\",\"type\":\"uint256\"},{\"name\":\"value\",\"type\":\"uint256\"}]},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"type\":\"uint256\"}],\"stateMutability\":\"view\"}]"
  }
}
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "name": "MultiToken",
    "address": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "status": "registered"
  }
}
```

#### Result Fields

| Field | Type | Description |
|-------|------|-------------|
| name | string | The registered name (echo of the request) |
| address | string | The registered contract address |
| status | string | Always `"registered"` on success |

#### Errors

| When | Code | Message |
|------|------|---------|
| `address` or `abi` missing | -32600 | `address and abi are required` |
| ABI fails to parse/validate | -32600 | *(parser error describing the invalid ABI)* |
| Registry not configured | -32000 | `contract registry not configured` |

> **Tuple/struct outputs** in `view` methods are not yet supported by the
> importer and such an ABI is rejected. Events with tuple **inputs** are fully
> supported. (See the limitations note under `contractCall`.)

---

## contractSubscribe

🔒 **Secured.** Subscribes an existing service to a registered contract's
events **and** transactions — one call, both notification types. From this
point, every event that matches your `scope` is pushed to your configured
`eventUrl` as a `contractEvent` notification, and (for `whole_contract` scope)
every transaction sent to the contract is pushed as a `contractTransaction`
notification — see [Scope and contractTransaction](#event-scopes).
Subscriptions are **persistent** and resume after a restart.

**Method:** `contractSubscribe` (alias `contract.subscribe`)

#### Parameters

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| serviceId | int | ✅ | The service that will receive the events (also used for auth) |
| address | string | ✅ | The contract address to subscribe to (**must already be registered**) |
| scope | string | ✅ | `whole_contract` or `managed_only` (see [Event scopes](#event-scopes)) |
| selectors | array of string | optional | Raw 4-byte method selectors, `0x`-hex (e.g. `"0xa9059cbb"`) — narrows `contractTransaction` to only these operations. See [Filtering contractTransaction by operation](#filtering-contracttransaction-by-operation). |
| methods | array of string | optional | Canonical method signatures (e.g. `"transfer(address,uint256)"`) — resolved to their selector the same way `selectors` works, just more ergonomic when you know the signature by name. May be combined with `selectors`. |

Neither `selectors` nor `methods` given means **no filter**: every transaction
sent to the contract is delivered, which is also today's behavior for a
caller that never uses this. `contractEvent` is never filtered by selector —
only `contractTransaction`.

#### Request Example — plain subscribe (no operation filter)
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "contractSubscribe",
  "params": {
    "serviceId": 42,
    "address": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "scope": "managed_only"
  }
}
```

#### Request Example — filtered to specific operations
```json
{
  "id": 2,
  "jsonrpc": "2.0",
  "method": "contractSubscribe",
  "params": {
    "serviceId": 42,
    "address": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "scope": "whole_contract",
    "methods": ["swap(uint256,uint256,uint256,uint256,address)"],
    "selectors": ["0xa9059cbb"]
  }
}
```
This subscription's `contractTransaction` notifications are limited to calls
matching either `swap(...)`'s signature or the raw selector `0xa9059cbb` —
nothing else sent to the contract is delivered.

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "serviceId": 42,
    "address": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "scope": "managed_only",
    "status": "subscribed"
  }
}
```

#### Errors

| When | Code | Message |
|------|------|---------|
| `serviceId` or `address` missing | -32600 | `serviceId and address are required` |
| Contract not registered | -32600 | `unknown contract: register its ABI before subscribing` |
| Invalid `scope` | -32600 | `unknown event scope "…" (want whole_contract \| managed_only)` |
| Malformed entry in `selectors` | -32600 | `selectors: "…": …` (not valid hex, or not exactly 4 bytes) |
| Event subscriber not configured | -32000 | `event subscriber not configured` |

---

### Filtering contractTransaction by operation

By default a `whole_contract` subscription's `contractTransaction` fires for
**every** transaction sent to the contract — noisy for a busy contract like a
DEX pool, where a subscriber usually cares about one kind of call (`swap`,
say) and not routine admin calls or calls to every other method. `selectors`
and `methods` narrow delivery to only the operations named.

**Why by selector, not by method name.** A filter is matched against the
transaction's raw 4-byte method selector (the first 4 bytes of its calldata),
never against the decoded method name. Two differently-typed overloads of one
name — `transfer(address,uint256)` and `transfer(address,uint256,bytes)` — are
two different selectors; naming the bare string `"transfer"` would be
ambiguous between them, so it is not accepted. Name a full signature (via
`methods`) or the selector itself (via `selectors`) instead.

**Why both forms exist.** `selectors` needs no ABI lookup at all — it works
even for a method this node's registered ABI does not define, as long as you
know the 4 bytes observed on-chain. `methods` is resolved to a selector the
same way — a pure keccak256 hash of the signature string, no ABI lookup
either — but is more ergonomic when you know the signature by name rather
than its hash. Use either, or both together; they merge into one filter set.

**Interaction with undecodable transactions.** A transaction whose selector
matches the filter is delivered even if this node's registered ABI cannot
decode it (see the `contractTransaction` unknown-method case) — filtering
happens on the raw selector bytes, before decoding is attempted. A plain
value transfer (no calldata at all) never matches a filtered subscription,
since there is no selector to filter on; an **unfiltered** subscription still
receives it, unchanged.

**Not yet supported:** filtering `contractEvent` by event name/topic, and
`managed_only` for `contractTransaction` — see the Future Phases note in
`todo/TASKS.md`.

---

## contractUnsubscribe

🔒 **Secured.** Removes a service's subscription to a contract — stops both
`contractEvent` and `contractTransaction` delivery for it. Idempotent —
removing a non-existent subscription is a no-op success.

**Method:** `contractUnsubscribe` (alias `contract.unsubscribe`)

#### Parameters

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| serviceId | int | ✅ | The subscribed service (also used for auth) |
| address | string | ✅ | The contract address to unsubscribe from |

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "contractUnsubscribe",
  "params": {
    "serviceId": 42,
    "address": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380"
  }
}
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "serviceId": 42,
    "address": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "status": "unsubscribed"
  }
}
```

---

## contractList

**Open.** Returns all registered contracts as a `name → address` map.

**Method:** `contractList` (alias `contract.list`)

#### Parameters
None.

#### Request Example
```json
{ "id": 1, "jsonrpc": "2.0", "method": "contractList" }
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "MultiToken": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "USDT": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380"
  }
}
```

#### Result

A JSON object mapping each registered contract's **name** to its **address**.

---

## contractSubscriptions

**Open.** Returns all active event subscriptions across all services.

**Method:** `contractSubscriptions` (alias `contract.subscriptions`)

#### Parameters
None.

#### Request Example
```json
{ "id": 1, "jsonrpc": "2.0", "method": "contractSubscriptions" }
```

#### Response Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": [
    {
      "serviceId": "42",
      "address": "0x3b5e7b8ac801ea77077b889fa7a778abcba38380",
      "scope": "managed_only",
      "selectors": []
    },
    {
      "serviceId": "42",
      "address": "0x9c5083dd4a6e1b2e4a3a2a5f2f2f2f2f2f2f2f2f",
      "scope": "whole_contract",
      "selectors": ["0xa9059cbb"]
    }
  ]
}
```

#### Result Fields (per entry)

| Field | Type | Description |
|-------|------|-------------|
| serviceId | string | The subscribed service id (as a string) |
| address | string | The contract address (stored lowercased) |
| scope | string | `whole_contract` or `managed_only` |
| selectors | array of string | The `contractTransaction` operation filter, as raw `0x`-hex selectors (whatever mix of `selectors`/`methods` was given at subscribe time, always resolved to this one form). Empty means no filter — see [Filtering contractTransaction by operation](#filtering-contracttransaction-by-operation). |

---

## contractCall

**Open.** Calls a **read-only** (`view` / `pure`) method of a registered
contract via `eth_call` and returns the **decoded** outputs. Use this to read
on-demand state such as `totalSupply`, `decimals`, `name`, `symbol`, etc.

**Method:** `contractCall` (alias `contract.call`)

#### Parameters

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| address | string | ✅ | The contract address (must be registered) |
| method | string | ✅ | The view method name to call (e.g. `totalSupply`) |
| args | array | optional | Positional arguments — **see limitation below** |

> ⚠️ **First-cut limitation — no-argument methods only.** Passing a non-empty
> `args` array currently returns the error `contractCall with arguments is not
> yet supported`. This is deliberate: typed argument encoding from JSON
> (`uint256`, `address`, tuples, …) is a planned follow-up, and the service
> refuses rather than risk mis-encoding. No-arg view methods (`totalSupply`,
> `decimals`, `name`, `symbol`, …) work today.

#### Request Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "contractCall",
  "params": {
    "address": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "method": "totalSupply"
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
      "type": "uint256",
      "value": "100000000000000000000"
    }
  ]
}
```

#### Result

An **array of decoded values** in the method's declared output order. Each
element follows the **[Decoded value format](#decoded-value-format)** below.
Output values carry their declared `name` when the ABI names them.

#### Errors

| When | Code | Message |
|------|------|---------|
| `address` or `method` missing | -32600 | `address and method are required` |
| `args` provided | -32600 | `contractCall with arguments is not yet supported` |
| Contract caller not configured | -32000 | `contract caller not configured` |
| Call/decoding failed | -32600 | *(underlying error, e.g. unknown contract / method)* |

---

## contractEvent

**Notification (push).** Delivered to a subscriber's configured `eventUrl` over
HTTP POST as a JSON-RPC 2.0 notification, whenever a decoded contract log event
matches that subscriber's subscription `scope`. This is the asynchronous
counterpart to the on-demand `contractCall`.

Delivery uses the **same webhook mechanism** as `blockEvent` / `transactionEvent`
(see [Events & Webhooks](#events--webhooks) for the delivery model, ordering, and
at-least-once semantics).

#### Notification Example
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "contractEvent",
  "params": {
    "event": "TransferSingle",
    "contract": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "blockNum": 20123456,
    "txHash": "0x4b1edb1329619c67467fb916a0b78938eb878078ac59ba9afdd7a34b0646e02e",
    "txIndex": 2,
    "logIndex": 7,
    "inputs": [
      { "name": "operator", "type": "address", "value": "0x101112131415161718191a1b1c1d1e1f20212223" },
      { "name": "from",     "type": "address", "value": "0xa0a1a2a3a4a5a6a7a8a9aaabacadaeafb0b1b2b3" },
      { "name": "to",       "type": "address", "value": "0x101112131415161718191a1b1c1d1e1f20212223" },
      { "name": "id",       "type": "uint256", "value": "7" },
      { "name": "value",    "type": "uint256", "value": "1000000000000000000" }
    ]
  }
}
```

#### Notification Parameters

| Field | Type | Description |
|-------|------|-------------|
| event | string | The event name (e.g. `Transfer`, `TransferSingle`) |
| contract | string | The emitting contract address (as registered) |
| blockNum | int64 | Block number containing the log |
| txHash | string | Transaction hash that produced the log |
| txIndex | int64 | Index of the transaction within the block |
| logIndex | int64 | Index of the log within the block |
| removed | bool | *(present only when `true`)* the log was reverted by a chain reorg |
| inputs | array | The decoded event parameters in ABI order — see [Decoded value format](#decoded-value-format) |

#### Notes

- An event is delivered **once per matching subscription**. If two services
  subscribe to the same contract, each receives its own `contractEvent`.
- For `managed_only` subscriptions, only events involving a managed address are
  delivered (matched on the decoded `address`-typed parameters).
- **Reorgs:** if a previously delivered log is reverted, a follow-up event with
  `"removed": true` may be delivered. Treat `removed` events as a revert signal.
- Delivery is **at-least-once**; deduplicate using
  `(txHash, logIndex)` which is unique per log.

---

## contractTransaction

**Notification (push).** Delivered to a subscriber's configured `eventUrl` over
HTTP POST as a JSON-RPC 2.0 notification, for **every transaction sent
directly to** a subscribed contract (`to` equals the contract address) —
regardless of whether it emitted any event, and regardless of whether it
reverted. A method call with no logs, or one that failed, is invisible to
`contractEvent`; `contractTransaction` is how you see it.

Delivered from the **same** subscription as `contractEvent`
(`{serviceId, address, scope}` from `contractSubscribe`) — there is no
separate subscribe step. Currently only `whole_contract` subscriptions receive
it; see [Scope and contractTransaction](#event-scopes).

Delivery uses the **same webhook mechanism** as `blockEvent` /
`transactionEvent` / `contractEvent` (see
[Events & Webhooks](#events--webhooks) for the delivery model, ordering, and
at-least-once semantics).

#### Notification Example — decoded call
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "contractTransaction",
  "params": {
    "contract": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "blockNum": 20123456,
    "txHash": "0x4b1edb1329619c67467fb916a0b78938eb878078ac59ba9afdd7a34b0646e02e",
    "from": "0xa0a1a2a3a4a5a6a7a8a9aaabacadaeafb0b1b2b3",
    "to": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "value": "0",
    "gas": 100000,
    "gasUsed": 51423,
    "success": true,
    "method": "transfer",
    "inputs": [
      { "name": "to", "type": "address", "value": "0x101112131415161718191a1b1c1d1e1f20212223" },
      { "name": "amount", "type": "uint256", "value": "1000000000000000000" }
    ],
    "data": "0xa9059cbb0000000000000000000000001011121314151617181920212223...000de0b6b3a7640000"
  }
}
```

#### Notification Example — unknown method
The same shape, when the transaction's method selector matches nothing in the
contract's registered ABI. `method` carries the raw 4-byte selector instead of
a name, `inputs` is empty, and `data` (always present) is the only way to see
what was actually sent:
```json
{
  "method": "contractTransaction",
  "params": {
    "contract": "0x3B5E7b8ac801EA77077b889fa7A778ABcBa38380",
    "...": "... same fields as above ...",
    "method": "0xa9059cbb",
    "inputs": [],
    "data": "0xa9059cbb0000000000000000000000001011121314151617181920212223...000de0b6b3a7640000"
  }
}
```

#### Notification Parameters

| Field | Type | Description |
|-------|------|-------------|
| contract | string | The called contract address (as registered) — same as `to` |
| blockNum | int64 | Block number containing the transaction |
| txHash | string | The transaction's hash |
| from | string | Sender address |
| to | string | Recipient address (the subscribed contract) |
| value | string | Native-coin amount moved, in wei, as a **decimal string** (not a JSON number — see [Decoded value format](#decoded-value-format) for why) |
| gas | int64 | Gas limit the transaction was sent with |
| gasUsed | int64 | Gas actually consumed |
| success | bool | `true` if the transaction succeeded, `false` if it reverted |
| method | string | The decoded method name; the raw 4-byte selector as `0x`-hex (e.g. `"0xa9059cbb"`) if it matches no registered method; `""` if the transaction carried no calldata at all (a plain value transfer) |
| inputs | array | The decoded call arguments in ABI order, empty when `method` did not decode — see [Decoded value format](#decoded-value-format) |
| data | string | The raw calldata as `0x`-hex, **always present** regardless of whether `method`/`inputs` decoded — decode it yourself, or use it to double-check a decoded call |

#### Notes

- Delivered **once per matching subscription, per transaction**. If two
  services subscribe `whole_contract` to the same contract, each receives its
  own `contractTransaction`.
- A reverted transaction (`success: false`) is delivered like any other — it
  still called the contract and still cost gas. It is not treated as an error.
- `managed_only` subscriptions do **not** receive `contractTransaction` today
  — see [Scope and contractTransaction](#event-scopes).
- Delivery is **at-least-once**; deduplicate using `txHash`, which is unique
  per transaction (unlike `contractEvent`, there is no `logIndex` — a
  transaction produces at most one `contractTransaction` per subscription).

---

## Decoded value format

`contractEvent.inputs[]`, `contractTransaction.inputs[]`, and `contractCall`
results are all arrays of **decoded values**. Every decoded value has this
shape:

```json
{ "name": "value", "type": "uint256", "value": <encoded> }
```

| Field | Type | Description |
|-------|------|-------------|
| name | string | The parameter/output name from the ABI (**omitted** if the ABI does not name it) |
| type | string | The ABI type (`address`, `uint256`, `bool`, `string`, `bytes`, `bytes32`, `uint256[]`, `tuple`, …) |
| value | varies | The decoded value, **encoded JSON-safely** as described below |

### How `value` is encoded (read this for safe integration)

The wire encoding is chosen so that **every value is safe to parse in any
language, including JavaScript** — no precision loss, no base64 surprises:

| ABI type | `value` JSON form | Example |
|----------|-------------------|---------|
| `uintN`, `intN` (incl. `uint256`) | **decimal STRING** | `"1000000000000000000"` |
| `address` | **`0x`-prefixed lowercase hex** string (20 bytes) | `"0xa0a1…b2b3"` |
| `bytesN`, `bytes` | **`0x`-prefixed lowercase hex** string (empty → `"0x"`) | `"0xdeadbeef"` |
| `bool` | native JSON boolean | `true` |
| `string` | native JSON string | `"hello"` |
| `T[]`, `T[N]` (arrays) | JSON **array** of decoded values | `[{ "type":"uint256","value":"10" }, …]` |
| `tuple` / struct | JSON **array** of decoded values (one per component) | `[{ "type":"address","value":"0x…" }, …]` |

> 🔑 **Integers are strings, not numbers.** `uint256` values routinely exceed
> JavaScript's `Number.MAX_SAFE_INTEGER` (2⁵³). They are delivered as **decimal
> strings** so `JSON.parse` never silently corrupts them. Parse them with a
> big-integer type (`BigInt(value)` in JS, `int`/`Decimal` in Python, etc.).

> 🔑 **Byte types are `0x`-hex, not base64.** Addresses and byte values are
> always `0x`-prefixed lowercase hex strings, ready to use directly.

#### Indexed reference-type parameters

Per the Ethereum ABI spec, an **indexed** parameter of a *reference* type
(`string`, `bytes`, arrays, or tuples) is stored in the log topics only as its
`keccak256` **hash**, not its original value — the value is **not recoverable**
from the log. For such parameters:

- `type` carries a ` (indexed)` suffix (e.g. `"string (indexed)"`) — this is a
  **display marker only**, not a valid ABI type string.
- `value` is the 32-byte hash as a `0x`-hex string.

Indexed *value* types (`address`, `uintN`, `intN`, `bool`, `bytesN`) are decoded
to their real values normally.
