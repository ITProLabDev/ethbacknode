package ethclient

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/crypto"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// signTransfer builds and signs a native-coin transfer in whichever envelope
// fees names -- EIP-1559 when fees.dynamic, legacy otherwise -- and returns
// the encoding eth_sendRawTransaction takes.
//
// Pure (no client/network access), so it is checked directly against known
// cast-verified vectors rather than only indirectly through a live send.
func signTransfer(pk *ecdsa.PrivateKey, chainID *big.Int, nonce int64, to []byte, amount *big.Int, fees txFees, gas int64) ([]byte, error) {
	if fees.dynamic {
		txSigner := crypto.NewEthDynamicFeeTxSigner(uint64(nonce), fees.maxPriorityFeePerGas, fees.maxFeePerGas, uint64(gas), to, amount, nil)
		txSigner.SetChainId(chainID)
		sign, err := txSigner.Sign(pk)
		if err != nil {
			log.Error("Can not sign transaction:", err)
			return nil, err
		}
		if len(sign) == 0 {
			log.Error("Can not sign transaction: sign is empty")
			return nil, ErrTransactionSignError
		}
		return txSigner.EncodeRPL()
	}

	txSigner := &crypto.EthTxSigner{
		Nonce:    uint64(nonce),
		GasPrice: fees.gasPrice,
		Gas:      uint64(gas),
		To:       &to,
		Value:    amount,
	}
	txSigner.SetChainId(chainID)
	sign := txSigner.Sign(pk)
	if len(sign) == 0 {
		log.Error("Can not sign transaction: sign is empty")
		return nil, ErrTransactionSignError
	}
	return txSigner.EncodeRPL()
}

func (c *Client) sendRawByPrivateKeyUnsafe(fromPrivateKey []byte, from, to string, amount *big.Int, fees txFees, gas int64) (txHash string, err error) {
	toBytes, err := c.addressCodec.DecodeAddressToBytes(to)
	if err != nil {
		return "", err
	}
	pk, _ := crypto.ECDSAKeysFromPrivateKeyBytes(fromPrivateKey)

	return c.withAllocatedNonce(from, func(nonce int64) (string, error) {
		netId, err := c.GetNetId()
		if err != nil {
			return "", err
		}
		chainID := big.NewInt(netId)
		log.Warning("ChainID:", chainID)

		txSignedBytes, err := signTransfer(pk, chainID, nonce, toBytes, amount, fees, gas)
		if err != nil {
			return "", err
		}

		log.Warning("txSignedBytes", hexnum.BytesToHex(txSignedBytes))
		hash, err := c.SendRawTransaction(hexnum.BytesToHex(txSignedBytes))
		if err != nil {
			log.Error("Can not broadcast transaction:", err)
			return "", err
		}
		return hash, nil
	})
}
