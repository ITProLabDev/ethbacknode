# Tron network support — research findings & required work

**Status:** research only, no code written. Captures findings from a session
on 2026-09-08 so the work can be picked up without repeating the discovery.

## Goal

Add a Tron client to ethbacknode, alongside the existing Ethereum client,
while preserving the project's existing architecture and conventions (own
crypto/ABI engine, `types.ChainClient` abstraction, `address.AddressCodec`
abstraction — see `[[own-abi-no-eth-libs]]`-style constraints already
enforced for the Ethereum side).

## Prior art found

Three related local projects, all under `~/go/src/git.voodoocore.io/core/`:

- **`ethconn`** — the direct prototype ethbacknode evolved from (Ethereum-only
  backend service).
- **`tronconn`** — an analogous standalone Tron-only backend service, same
  author/vintage as ethconn, but structured completely differently from it
  (`tronclient/accountmethods.go`, `blockmethods.go`, `coremethods.go`,
  `smatrcontractmethods.go`, `tron_client.go`, ...) — organized around Tron's
  own API groupings, not mirrored against ethclient's shape at all.
- **`uniconn`** (found during this research, not initially known to the
  user) — a **third** project that already unified Ethereum and Tron clients
  behind one shared `ChainClient` interface. This is the most directly
  relevant prior art:
  - `uniconn/interfaces.go:9-58` defines `ChainClient`, composed of the same
    7 sub-interfaces ethbacknode's own `types.ChainClient` has
    (`ChainClientInfo`/`Blocks`/`MemPool`/`Transactions`/`Balances`/
    `CoinTransfer`/`TokenTransfer`) — method-for-method identical, just
    against `unitypes.*` instead of `types.*`.
  - `uniconn/clients/tronclient/` is a **deliberate rewrite** of tronconn's
    tronclient, restructured to mirror `uniconn/clients/ethclient/`'s file
    layout exactly (`client.go`, `config.go`, `decode_block.go`,
    `decode_transaction.go`, `errors.go`, `methods.go`, `options.go`,
    `send_methods.go`, `tokens.go`, `types_block.go`,
    `types_transaction.go`, `address_encoder.go`) — not a superficial copy.
  - **Caveat:** stale (last commit 2024-07-30, over a year old at time of
    writing) and incomplete — **zero test files**, and six methods are
    `panic("implement me")` stubs (`TokensBalanceOf`, `BlockByHash`,
    `TransferInfoByNum`, and all three token-transfer methods). Useful as an
    architectural blueprint, not as drop-in code.

## Why this looks tractable: address handling is (almost) already solved

Tron addresses are derived **exactly the same way** as Ethereum addresses —
`keccak256(uncompressed pubkey[1:])[12:]`, the same secp256k1 curve, the same
signing primitives. The only difference is the final string encoding:
EIP-55 checksummed hex (Ethereum) vs. `base58check(0x41 ‖ addressBytes)`
(Tron). Confirmed in `uniconn`: `ethclient.AddressCodec.PrivateKeyToAddress`
and `tronclient.AddressCodec.PrivateKeyToAddress`
(`uniconn/clients/tronclient/address_encoder.go:20-25`) are byte-identical up
to the encoding step.

ethbacknode already has every primitive this needs, with no new dependency:

- `address.AddressCodec` (`address/address.go:45-54`) is already
  chain-agnostic in shape: `EncodeBytesToAddress` / `DecodeAddressToBytes` /
  `PrivateKeyToAddress` / `IsValid`.
- `common/base58` already exists, including `base58check.go`.
- `crypto.PubKeyToAddressBytes` (`crypto/ecdsa.go:18`) already does the
  keccak-based derivation Tron needs.

So a `TronAddressCodec` implementing `address.AddressCodec` is plausibly a
small, self-contained new file, not a new subsystem.

## Where it's genuinely different — the real work

1. **Transaction construction & signing.** Tron's model differs
   structurally, not just cosmetically:
   - The signing digest is **SHA-256** of the transaction's raw bytes, not
     Keccak256.
   - No nonce/account-sequence; replay protection is a `ref_block` hash +
     expiration timestamp scheme instead.
   - Fees are priced by **bandwidth** (roughly, serialized transaction size)
     rather than gas.
   - `uniconn`'s own `send_methods.go` gets the unsigned transaction bytes
     from the node's `wallet/createtransaction` endpoint rather than building
     them locally — a `// TODO !!! prepare transaction without call java
     tron` marks that local (node-independent) transaction construction was
     never finished, even there.
   - Given this project's own standard (see the EIP-1559 signer built and
     verified against `cast` this session, `crypto/eth_dynamic_fee_tx_signer.go`),
     the expectation would be a **local**, from-scratch, independently
     verified Tron transaction builder/signer — not a node-assisted
     shortcut — which is real, non-trivial new work, not a port.

2. **Transport.** Tron nodes expose plain HTTP+JSON REST-ish endpoints
   (`/wallet/getnowblock`, `/wallet/broadcasttransaction`, ...), **not**
   JSON-RPC 2.0 like Ethereum. `uniconn` handled this by adding a second mode
   to its shared transport package rather than a bespoke client
   (`tronclient/options.go:18`: `urpc.WithHTTPRest(endpointUrl, headers)`
   alongside `ethclient`'s `WithHTTPRpc`). ethbacknode's own `clients/urpc`
   would need the analogous extension: a REST-style call mode alongside the
   existing JSON-RPC 2.0 one.

3. **`main.go` is hard-wired to Ethereum today.** The chain client is
   constructed once, unconditionally, as `ethclient.NewClient(...)`
   (`main.go:141`), and the `--init` chain-bootstrap flag resolves profiles
   only through `ethclient.ChainProfileByName` (`init.go:26`). Adding Tron
   needs a selection point: which concrete client to construct (an
   `ethclient.Client` or a new `tronclient.Client`, both satisfying
   `types.ChainClient`) and how `--init` picks a Tron profile alongside the
   existing `Eth`/`Arc` ones.

4. **`abi/` reuse — plausible, not verified.** TVM (Tron's contract VM) is
   generally ABI-compatible with EVM for contract call encoding/decoding, so
   ethbacknode's own custom `abi/` engine may work for Tron smart contracts
   largely as-is. This was **not verified** during this research — flagged
   as an open question, not a finding.

## Open questions for the user (not yet decided)

- **Single process vs. separate deployments.** `ethconn`/`tronconn` were
  historically separate services. Does "preserving compatibility" mean one
  ethbacknode binary running both chains at once (would touch `watchdog`,
  `eventlog`, `subscriptions`, `endpoint` — all currently assume one
  `chainClient`), or a Tron-capable build/config of the same codebase run as
  its own instance per chain (much smaller blast radius)?
- **Scope/order to start with:** address codec first (small, self-contained,
  testable immediately against known Tron vectors), transport second, full
  client + signer last — or a different order.

## Next step

Not started. Revisit this file before beginning implementation; it is the
record of what was already learned so the research above does not need to be
repeated.
