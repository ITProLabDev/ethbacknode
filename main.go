// Package main implements the EthBackNode application - an Ethereum blockchain
// backend adapter that provides JSON-RPC 2.0 API for address management,
// transaction monitoring, and cryptocurrency transfers.
package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/clients/ethclient"
	"github.com/ITProLabDev/ethbacknode/clients/txmanager"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/endpoint"
	"github.com/ITProLabDev/ethbacknode/eventlog"
	"github.com/ITProLabDev/ethbacknode/presets"
	"github.com/ITProLabDev/ethbacknode/security"
	"github.com/ITProLabDev/ethbacknode/storage"
	"github.com/ITProLabDev/ethbacknode/subscriptions"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"github.com/ITProLabDev/ethbacknode/txcache"
	"github.com/ITProLabDev/ethbacknode/watchdog"
)

// Application version and identification constants.
const (
	// APP_VERSION is the current version of the application.
	APP_VERSION = "0.1.3dev"
	// CHAIN_NAME identifies the blockchain type (EVM-compatible).
	CHAIN_NAME = "EVM"
	// APP_NAME is the application name used for logging and identification.
	APP_NAME = "EthBackNode"
)

// Global variables for application state and configuration.
var (
	// globalConfigPath is the path to the configuration file (default: config.hcl).
	globalConfigPath = "config.hcl"
	// initChain holds the --init chain profile name (empty = normal startup).
	initChain string
	// config holds the global application configuration.
	config = &Config{
		storage: _configDefaultStorage(),
	}

	// done is a channel used to signal application shutdown.
	done = make(chan bool)
	// osSig is a channel that receives OS signals for graceful shutdown.
	osSig = make(chan os.Signal, 1)
)

// main is the application entry point. It initializes all subsystems in the following order:
// 1. Storage manager - data persistence layer
// 2. ABI manager - smart contract ABI registry
// 3. Chain client - Ethereum node connection (HTTP-RPC or IPC)
// 4. Address manager - address generation and pool management
// 5. Watchdog service - blockchain monitoring
// 6. Subscriptions manager - event notification system
// 7. Transaction cache - transaction caching
// 8. Security manager - API authentication
// 9. RPC endpoint server - JSON-RPC 2.0 HTTP server
func main() {
	parseFlags()
	if initChain != "" {
		bootstrapAndExit() // generates config for the chain profile and exits
	}
	signal.Notify(osSig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	log.Info(APP_NAME, CHAIN_NAME, " Connection Adapter", APP_VERSION)
	configStorage, err := storage.NewBinFileStorage("Config", "", "", globalConfigPath)
	if err != nil {
		log.Error("Can not get init config storage:", err)
		os.Exit(-1)
	}
	config.storage = configStorage
	err = config.Load()
	if err != nil {
		log.Error("Can not load config:", err)
		os.Exit(-1)
	}
	log.Info("Node connection settings:")
	if !config.NodeUseIPC {
		log.Info("- Node Connection : http-rpc")
		log.Info("- Node Url        :", config.NodeUrl)
		log.Info("- Node Port       :", config.NodePort)
	} else {
		log.Info("- Node Connection: ipc socket")
		log.Info("- Node ipc Socket Path:", config.NodeIPCSocket)
	}
	addressCodec := ethclient.GetAddressCodec()
	// Init global storage manager
	storageManager, err := storage.NewStorageManager(config.DataPath)
	if err != nil {
		log.Error("Can not init storage manager:", err)
		os.Exit(-1)
	}
	// Get Address Codec
	// Init Smart Contract ABI manager
	abiStorage := storageManager.GetModuleStorage("ABI", "abi")
	abiManager := abi.NewManager(
		abi.WithStorage(abiStorage.GetBinFileStorage("known_contracts.json")),
		abi.WithAddressCodec(addressCodec),
	)
	err = abiManager.Init()
	if err != nil {
		log.Error("Can not load abi manager:", err)
		os.Exit(-1)
	}
	clientStorage := storageManager.GetModuleStorage("Client", "client")
	var clientOptions = []ethclient.Option{
		ethclient.WithConfigStorage(clientStorage.GetBinFileStorage("config.json")),
		ethclient.WithAbiManager(abiManager),
	}
	if config.NodeUseIPC {
		// ipcPoolSize > 1 opens a pool of IPC connections so independent node
		// RPC calls run in parallel instead of serializing on one socket.
		// Defaults to 1 (single connection) for backward compatibility.
		ipcPoolSize := config.Int("ipcPoolSize", 1)
		if ipcPoolSize > 1 {
			log.Info("- Node IPC connection pool size:", ipcPoolSize)
			clientOptions = append(clientOptions,
				ethclient.WithIPCClientPool(config.NodeIPCSocket, ipcPoolSize, 30*time.Second))
		} else {
			clientOptions = append(clientOptions, ethclient.WithIPCClient(config.NodeIPCSocket))
		}
	} else {
		clientOptions = append(clientOptions,
			ethclient.WithRpcClient(
				config.NodeUrl,
				config.NodePort,
				config.NodeUseSSl,
				config.AdditionalHeaders,
			),
		)
	}
	chainClient := ethclient.NewClient(clientOptions...)
	err = chainClient.Init()
	if err != nil {
		log.Error("Can not init chain client:", err)
		os.Exit(-1)
	}
	log.Info("Blockchain Info:")
	log.Info("- Chain Name:", chainClient.GetChainName())
	log.Info("- Chain ID:", chainClient.GetChainId())
	// Apply any built-in contract preset for this chain (PROJECT_STATUS §6).
	// Matched on the numeric chainId from eth_chainId — the on-chain truth and
	// the key the deployment artifact uses. Non-fatal: a node without the preset
	// behaves as before. Idempotent: the registry dedups by address on restart.
	if netId, nerr := chainClient.GetNetId(); nerr != nil {
		log.Error("Can not read chain id for contract preset:", nerr)
	} else if loaded, perr := presets.Apply(netId, abiManager); perr != nil {
		log.Error("Can not apply contract preset:", perr)
	} else if loaded > 0 {
		log.Info("- Loaded contract preset:", loaded, "contracts")
	}
	for _, token := range chainClient.TokensList() {
		log.Info("- Token:", token.Name, "(", token.Symbol, ")")
	}
	// Nonce allocator: the client and the manager are mutually dependent at
	// construction (the manager seeds its cursor from the client, the client
	// takes its nonces from the manager), so the client is completed with
	// SetNonceSource only after both exist. Without this, the client falls
	// back to asking the node directly on every send, which is correct for
	// one transaction at a time and wrong for two prepared back to back.
	txManagerStorage := storageManager.GetModuleStorage("TxManager", "txmanager")
	nonceManager := txmanager.NewNonceManager(
		txmanager.WithChain(chainClient),
		txmanager.WithStore(txmanager.NewBadgerHoldStore(txManagerStorage.GetNewBadgerHoldStorage("nonces.db"))),
	)
	chainClient.SetNonceSource(nonceManager)
	// Init Address Manager
	addressStorage := storageManager.GetModuleStorage("Address", "address")
	addressManager, err := address.NewManager(
		address.WithAddressCodec(addressCodec),
		address.WithConfigStorage(addressStorage.GetBinFileStorage("config.json")),
		address.WithAddressStorage(addressStorage.GetNewBadgerStorage("addresses.db")),
	)
	if err != nil {
		log.Error("Can not init address manager:", err)
		os.Exit(-1)
	}
	if config.DebugMode {
		addressManager.DevDumpMemPool()
	}
	// Init Watchdog Service
	watchdogStorage := storageManager.GetModuleStorage("Watchdog", "watchdog")
	watchDogOptions := []watchdog.ServiceOption{
		watchdog.WithConfigStorage(watchdogStorage.GetBinFileStorage("config.json")),
		watchdog.WithStateStorage(watchdogStorage.GetBinFileStorage("state.json")),
		watchdog.WithClient(chainClient),
		watchdog.WithAddressManager(addressManager),
	}
	if config.DebugMode {
		log.Warning("DEBUG MODE: reset watchdog last state to 0 block")
	}
	watchdogService := watchdog.NewService(watchDogOptions...)
	subscriptionsStorage := storageManager.GetModuleStorage("Subscriptions", "subscriptions")
	subscriptionsManager, err := subscriptions.NewManager(
		subscriptions.WithAddressManager(addressManager),
		subscriptions.WithSubscribersStorage(subscriptionsStorage.GetBinFileStorage("subscribers.json")),
		subscriptions.WithTransactionStorage(subscriptionsStorage.GetNewBadgerHoldStorage("transactions.db")),
		subscriptions.WithBlockchainClient(chainClient),
		subscriptions.WithConfigStorage(subscriptionsStorage.GetBinFileStorage("config.json")),
		subscriptions.WithGlobalConfig(config),
	)

	if err != nil {
		log.Error("Can not init subscriptions manager:", err)
		os.Exit(-1)
	}

	txCacheStorage := storageManager.GetModuleStorage("TxCache", "txcache")

	txCacheManager, err := txcache.NewManager(
		txcache.WithConfigStorage(txCacheStorage.GetBinFileStorage("config.json")),
		txcache.WithTxStorage(txCacheStorage.GetNewBadgerHoldStorage("txcache.db")),
	)
	if err != nil {
		log.Error("Can not start transactions cache manager:", err)
		os.Exit(-1)
	}
	watchdogService.RegisterTransactionEventListen(subscriptionsManager.TransactionEvent)
	watchdogService.RegisterTransactionEventListen(txCacheManager.TransactionEvent)
	watchdogService.RegisterBlockEventListen(subscriptionsManager.BlockEvent)
	watchdogService.RegisterBlockEventListen(txCacheManager.BlockEvent)

	// Event-log service: decode registered contracts' events per block and
	// deliver them. Wired as a watchdog block listener (Approach A).
	eventlogStorage := storageManager.GetModuleStorage("EventLog", "eventlog")
	eventLogService := eventlog.New(
		eventlog.WithLogSource(eventlog.RawLogSource{
			GetLogsFn: func(fromBlock, toBlock int64, addresses, topics []string) ([]eventlog.RawLog, error) {
				logs, err := chainClient.GetLogs(ethclient.LogFilter{
					FromBlock: fromBlock, ToBlock: toBlock, Addresses: addresses, Topics: topics,
				})
				if err != nil {
					return nil, err
				}
				out := make([]eventlog.RawLog, len(logs))
				for i, l := range logs {
					out[i] = eventlog.RawLog{
						Address: l.Address, Topics: l.Topics, Data: l.Data,
						BlockNumber: l.BlockNumber, TransactionHash: l.TransactionHash,
						TransactionIndex: l.TransactionIndex, LogIndex: l.LogIndex, Removed: l.Removed,
					}
				}
				return out, nil
			},
			GetReceiptFn: func(txHash string) (*eventlog.RawReceipt, error) {
				r, err := chainClient.GetTransactionReceipt(txHash)
				if err != nil {
					return nil, err
				}
				out := make([]eventlog.RawLog, len(r.Logs))
				for i, l := range r.Logs {
					out[i] = eventlog.RawLog{
						Address: l.Address, Topics: l.Topics, Data: l.Data,
						BlockNumber: l.BlockNumber, TransactionHash: l.TransactionHash,
						TransactionIndex: l.TransactionIndex, LogIndex: l.LogIndex, Removed: l.Removed,
					}
				}
				return &eventlog.RawReceipt{Logs: out, Status: r.Success(), GasUsed: r.GasUsed}, nil
			},
		}),
		eventlog.WithDecoder(eventlog.NewDecoder(abiManager.DecodeLog)),
		eventlog.WithManaged(addressManager),
		eventlog.WithAddressCodec(addressCodec),
		eventlog.WithSubscriptionStorage(eventlogStorage.GetBinFileStorage("subscriptions.json")),
		eventlog.WithSink(endpoint.NewContractEventSink(
			endpoint.NotifierFunc(func(serviceID int64, subject string, payload interface{}) {
				subscriptionsManager.NotifySubscriberRaw(subscriptions.ServiceId(serviceID), subject, payload)
			}),
		)),
		// contractTransaction: every transaction sent directly to a subscribed
		// contract, decoded by method selector (todo/TASKS.md Milestone 8).
		// A separate data path from log collection above: fetches the whole
		// block's transactions rather than logs.
		eventlog.WithTransactionSource(eventlog.RawTransactionSource{
			GetBlockTransactionsFn: func(blockNum int64) ([]eventlog.RawTransaction, error) {
				block, err := chainClient.GetBlockByNumber(blockNum, true)
				if err != nil {
					return nil, err
				}
				txs, err := block.GetTransactions()
				if err != nil {
					return nil, err
				}
				out := make([]eventlog.RawTransaction, len(txs))
				for i, t := range txs {
					input, ierr := hexnum.ParseHexBytes(t.Input)
					if ierr != nil {
						input = nil
					}
					out[i] = eventlog.RawTransaction{
						Hash: t.Hash, From: t.From, To: t.To, Value: t.Value, Gas: t.Gas, Input: input,
					}
				}
				return out, nil
			},
		}),
		eventlog.WithTransactionDecoder(eventlog.NewTransactionDecoder(abiManager.DecodeCall)),
		eventlog.WithTransactionSink(endpoint.NewContractTransactionSink(
			endpoint.NotifierFunc(func(serviceID int64, subject string, payload interface{}) {
				subscriptionsManager.NotifySubscriberRaw(subscriptions.ServiceId(serviceID), subject, payload)
			}),
		)),
		eventlog.WithConfig(eventlog.DefaultConfig()),
	)
	if err := eventLogService.LoadSubscriptions(); err != nil {
		log.Error("Can not load eventlog subscriptions:", err)
	}
	watchdogService.RegisterBlockEventListen(func(blockNum int64, blockId string) {
		eventLogService.OnBlock(blockNum, blockId)
	})

	log.Info("Init complete")
	err = watchdogService.Run()
	if err != nil {
		log.Error("Can not start watchdog service:", err)
		os.Exit(-1)
	}

	securityMaanger := security.NewManager(
		security.WithStorageManager(storageManager.GetModuleStorage("Security", "security")),
	)

	err = securityMaanger.Init()
	if err != nil {
		log.Error("Can not start security manager:", err)
		os.Exit(-1)
	}

	endpointRpcRouter := endpoint.NewBackRpc(
		addressManager,
		chainClient,
		subscriptionsManager,
		watchdogService,
		txCacheManager,
		endpoint.WithFallbackResponse(&endpoint.DevForm{
			FormPath: "dev/form.html",
		}),
		endpoint.WithDebugMode(config.DebugMode),
		endpoint.WithSecurityManager(securityMaanger),
		endpoint.WithAbiManager(abiManager, abiManager),
		endpoint.WithEventSubscriber(eventLogService),
		endpoint.WithContractCaller(chainClient),
	)
	endpointUrl, err := url.Parse(fmt.Sprintf("http://%s:%s", config.RpcAddress, config.RpcPort))
	if err != nil {
		log.Error("Can not parse endpoint url:", err)
		os.Exit(-1)
	}
	endpointServer, err := endpoint.NewServer(
		endpoint.WithHttpListener(endpointUrl.Host),
		endpoint.WithHandler(endpointRpcRouter.Handle),
	)
	if err != nil {
		log.Error("Can not init endpoint server:", err)
		os.Exit(-1)
	}
	log.Info("Start endpoint server on:", endpointUrl.Host)
	go func() {
		err = endpointServer.ListenAndServe()
		if err != nil {
			log.Error("Can not start endpoint server:", err)
			done <- true
		}
	}()
	// Start main loop
	run()
	log.Info("Application stopped")
}

// run is the main event loop that waits for shutdown signals.
// It listens on two channels:
// - done: internal shutdown signal (e.g., server error)
// - osSig: OS signals (SIGHUP, SIGINT, SIGTERM, SIGQUIT)
func run() {
	for {
		select {
		case _ = <-done:
			log.Info("Quit application by done...")
			return
		case _ = <-osSig:
			log.Warning("Quit application by OS Signal...")
			return
		}
	}
}

// parseFlags parses command-line flags. It is called from main() (NOT from an
// init() function) so that `go test` — which parses its own flags during
// startup — is not disrupted.
//
// Supported flags:
//   - config: path to configuration file (default: config.hcl)
//   - init:   bootstrap config for a chain profile (Eth | Arc) and exit
//   - help:   display usage information
func parseFlags() {
	var help bool
	flag.StringVar(&globalConfigPath, "config", "config.hcl", "Path to global config file")
	flag.StringVar(&initChain, "init", "", "Bootstrap config for a chain profile (Eth | Arc) and exit")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}
}

// bootstrapAndExit handles the --init flag: it generates config files for the
// selected chain profile (without overwriting existing ones) and exits. Called
// from main() only when initChain is non-empty.
func bootstrapAndExit() {
	mainStorage, err := storage.NewBinFileStorage("Config", "", "", globalConfigPath)
	if err != nil {
		log.Error("Can not open config storage:", err)
		os.Exit(-1)
	}
	clientStorage, err := storage.NewBinFileStorage("Config", "data", "client", "config.json")
	if err != nil {
		log.Error("Can not open client config storage:", err)
		os.Exit(-1)
	}
	res, err := runInit(initChain, mainStorage, clientStorage)
	if err != nil {
		log.Error("Init failed:", err)
		os.Exit(-1)
	}
	if res.WroteMain {
		log.Info("Wrote", globalConfigPath)
	} else {
		log.Info("Kept existing", globalConfigPath, "(not overwritten)")
	}
	if res.WroteClient {
		log.Info("Wrote data/client/config.json for chain:", initChain)
	} else {
		log.Info("Kept existing data/client/config.json (not overwritten)")
	}
	log.Info("Init complete for chain profile:", initChain)
	os.Exit(0)
}
