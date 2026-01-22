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

	"github.com/ITProLabDev/ethbacknode/abi"
	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/clients/ethclient"
	"github.com/ITProLabDev/ethbacknode/endpoint"
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
		clientOptions = append(clientOptions, ethclient.WithIPCClient(config.NodeIPCSocket))
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
	for _, token := range chainClient.TokensList() {
		log.Info("- Token:", token.Name, "(", token.Symbol, ")")
	}
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

// init parses command-line flags before main() is called.
// Supported flags:
//   - config: path to configuration file (default: config.hcl)
//   - help: display usage information
func init() {
	var help bool
	flag.StringVar(&globalConfigPath, "config", "config.hcl", "Path to global config file")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}
}
