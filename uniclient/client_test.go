package uniclient

import (
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"testing"
)

const (
	ADDRESS_MAIN = "0x74Fe1Af5df88AC160EfEf2F1559dACEe17EDD8F3"
	ADDRESS_TEST = ""
)

func TestClient(t *testing.T) {
	// Create a new client with the default options
	client := NewClient(
		WithHttpTransport("http://localhost:21280/rpc", nil),
		WithServiceId(42),
	)
	// Get a new address
	addressInfo, err := client.AddressGetNew(1, 0, false)
	if err != nil {
		t.Error("Error getting new address:", err)
	}
	log.Dump(addressInfo)
	// Get a new address
	addressInfo, err = client.AddressGetNewFullInfo(1, 0, false)
	if err != nil {
		t.Error("Error getting new address:", err)
	}
	log.Dump(addressInfo)
	// Get the balance for the address
	balance, err := client.BalanceGetForAddress(ADDRESS_MAIN, "ETH")
	if err != nil {
		t.Error("Error getting balance:", err)
	}
	log.Dump(balance)
	balances, err := client.BalanceGetForAddressAllAssets(ADDRESS_MAIN)
	if err != nil {
		t.Error("Error getting balance:", err)
	}
	log.Dump(balances)
	nodeInfo, err := client.GetNodeInfo()
	if err != nil {
		t.Error("Error getting node info:", err)
	}
	log.Dump(nodeInfo)
	fee, err := client.TransferGetEstimatedFee(ADDRESS_MAIN, ADDRESS_MAIN, "0.1", "ETH", true)
	if err != nil {
		t.Error("Error getting fee:", err)
	}
	log.Dump(fee)
	//0xf13ed48d5210fbc06eb4a397611db929cfce97ad702f8e8aba8525e1cf2fa3cc
	txInfo, err := client.TransferInfo("0xf13ed48d5210fbc06eb4a397611db929cfce97ad702f8e8aba8525e1cf2fa3cc")
	if err != nil {
		t.Error("Error getting tx info:", err)
	}
	log.Dump(txInfo)
	txList, err := client.TransfersByAddress(ADDRESS_MAIN)
	if err != nil {
		t.Error("Error getting tx list:", err)

	}
	log.Dump(txList)
	client.SetDebug(true)

}
