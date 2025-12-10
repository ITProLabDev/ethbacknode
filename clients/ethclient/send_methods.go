package ethclient

import (
	"backnode/common/hexnum"
	"backnode/crypto"
	"backnode/tools/log"
	"math/big"
)

func (c *Client) sendRawByPrivateKeyUnsafe(fromPrivateKey []byte, from, to string, amount, gasPrice *big.Int, gas int64) (txHash string, err error) {
	toBytes, err := c.addressCodec.DecodeAddressToBytes(to)
	if err != nil {
		return "", err
	}
	pk, _ := crypto.ECDSAKeysFromPrivateKeyBytes(fromPrivateKey)
	nonce, err := c.PendingNonceAt(from)
	if err != nil {
		return "", err
	}
	netId, err := c.GetNetId()
	if err != nil {
		return "", err
	}
	chainID := big.NewInt(netId)
	log.Warning("ChainID:", chainID)
	txSigner := &crypto.EthTxSigner{
		Nonce:    uint64(nonce),
		GasPrice: gasPrice,
		Gas:      uint64(gas),
		To:       &toBytes,
		Value:    amount,
	}
	txSigner.SetChainId(chainID)
	sign := txSigner.Sign(pk)
	if len(sign) == 0 {
		log.Error("Can not sign transaction: sign is empty")
		return "", ErrTransactionSignError
	}
	txSignedBytes, err := txSigner.EncodeRPL()
	if err != nil {
		log.Error("Can not encode signed transaction:", err)
		return "", err

	}
	log.Warning("txSignedBytes", hexnum.BytesToHex(txSignedBytes))
	txHash, err = c.SendRawTransaction(hexnum.BytesToHex(txSignedBytes))
	if err != nil {
		log.Error("Can not broadcast transaction:", err)
		return "", err
	}
	return txHash, nil
}
