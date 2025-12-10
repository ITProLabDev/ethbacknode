package endpoint

import (
	"encoding/json"
	"math/big"
	"module github.com/ITProLabDev/ethbacknode/common/hexnum"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"module github.com/ITProLabDev/ethbacknode/types"
	"strings"
)

type transferAmount json.Number

func (r *BackRpc) rpcProcessTransferAssets(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type transferAssetsRequest struct {
		ServiceID      int         `json:"serviceId,omitempty"`
		PrivateKey     string      `json:"privateKey,omitempty"`
		From           string      `json:"from,omitempty"`
		To             string      `json:"to"`
		Amount         json.Number `json:"amount"`
		Symbol         string      `json:"symbol,omitempty"`
		Force          bool        `json:"force,omitempty"`
		Signature      string      `json:"signature,omitempty"`
		AmountFormated bool        `json:"amountFormated,omitempty"`
	}
	var decimals int
	apiToken, err := ctx.GetApiToken()
	if err == nil {
		if r.debugMode {
			log.Warning("Api token found", apiToken)
		}
	}
	params := new(transferAssetsRequest)
	err = request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	params.From, err = r.addressNormalise(params.From)
	if err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid from address")
		return
	}
	params.To, err = r.addressNormalise(params.To)
	if err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid target address")
		return
	}
	if r.debugMode {
		log.Debug("Transfer assets request")
		log.Dump(params)
	}
	if params.From != "" && !r.addressCodec.IsValid(params.From) {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "from address required")
		return
	}
	if params.Symbol == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "asset symbol required")
		return
	}
	if params.To == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "to address required")
		return
	} else if !r.addressCodec.IsValid(params.To) {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid target address")
		return
	}
	if params.Symbol != r.chainClient.GetChainSymbol() && !r._isTokenKnown(params.Symbol) {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "unknown asset symbol")
		return
	}

	var amountToTransfer *big.Int
	var ok bool
	if params.AmountFormated {
		if params.Symbol == r.chainClient.GetChainSymbol() {
			decimals = r.chainClient.Decimals()
		} else {
			decimals = r.knownTokens[params.Symbol].Decimals
		}
		amountToTransfer, err = _parseAmountToBigInt(params.Amount.String(), decimals)
		if err != nil {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid amount")
			return
		}
	} else {
		amountToTransfer, ok = new(big.Int).SetString(params.Amount.String(), 10)
		if !ok {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid amount")
			return
		}
	}
	if amountToTransfer == nil || amountToTransfer.Sign() <= 0 || amountToTransfer.Cmp(big.NewInt(0)) == 0 {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid amount")
		return
	}
	transferData := &struct {
		PrivateKeyBytes []byte   `json:"privateKey,omitempty"`
		From            string   `json:"from,omitempty"`
		To              string   `json:"to"`
		Amount          *big.Int `json:"amount"`
		Symbol          string   `json:"symbol,omitempty"`
	}{}
	transferData.To = params.To
	transferData.Amount = amountToTransfer
	transferData.Symbol = params.Symbol
	//TODO check signature
	//TODO check serviceID
	transferData.From = params.From
	if params.PrivateKey != "" {
		if r.debugMode {
			log.Debug("Private key provided:", params.PrivateKey)
		}
		pkBytes, err := hexnum.ParseHexBytes(params.PrivateKey)
		if err != nil {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid private key")
			return
		}
		fromCalculated, _, err := r.chainClient.GetAddressCodec().PrivateKeyToAddress(pkBytes)
		if err != nil {
			if r.debugMode {
				log.Error("Can not calculate address from private key:", err)
			}
			response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid private key")
			return
		}
		if fromCalculated != params.From && params.From != "" {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "from address and key mismatch")
			return
		} else if fromCalculated != params.From {
			params.From = fromCalculated
		}
		transferData.PrivateKeyBytes = pkBytes
	} else if params.From == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "from address or private key required")
		return
	} else if params.PrivateKey == "" && params.From != "" {
		if r.debugMode {
			log.Debug("Private key not provided, using known address info")
		}
		if !r.addressPool.IsAddressKnown(params.From) {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "private key required")
			return
		}
		addressInfo, err := r.addressPool.GetAddress(params.From)
		if err != nil {
			log.Error("Can not get known address info: ", err)
			response.SetError(ERROR_CODE_SERVER_ERROR, ERROR_MESSAGE_SERVER_ERROR)
			return
		}
		if addressInfo.ServiceId != params.ServiceID {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "address unknown or not owned by service")
			return
		}
		if (addressInfo.WatchOnly && !params.Force) || len(addressInfo.PrivateKey) == 0 {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "address is watch only")
			return
		}
		transferData.PrivateKeyBytes = addressInfo.PrivateKey
	}
	var txHash string
	if transferData.Symbol == r.chainClient.GetChainSymbol() {
		if r.debugMode {
			log.Debug("Transferring native coin request")
		}
		txHash, err = r.chainClient.TransferByPrivateKey(transferData.PrivateKeyBytes, transferData.From, transferData.To, transferData.Amount)
	} else {
		if r.debugMode {
			log.Debug("Transferring token request")
		}
		txHash, err = r.chainClient.TransferTokenByPrivateKey(transferData.PrivateKeyBytes, transferData.From, transferData.To, transferData.Amount, transferData.Symbol)
	}
	if err != nil {
		//TODO check is it possible to get error from chain
		if r.debugMode {
			log.Error("Transfer error:", err)
		}
		response.SetError(ERROR_CODE_SERVER_ERROR, err.Error())
		return
	}
	result := &transferAssetsResult{}
	transferInfo, err := r.chainClient.TransferInfoByHash(txHash)
	if err != nil {
		log.Error("Can not get transfer info by hash:", err)
		result = &transferAssetsResult{
			TxID:    txHash,
			Success: false,
			Symbol:  transferData.Symbol,
			From:    transferData.From,
			To:      transferData.To,
			Amount:  amount(transferData.Amount.String()),
			Warning: "Can not get transfer info by hash",
		}
	} else {
		result.fill(transferInfo)
		if transferInfo.SmartContract {
			result.FeeSymbol = r.chainClient.GetChainSymbol()
		}
		if params.AmountFormated {
			result.Amount = amount(_formatBigIntToString(transferData.Amount, decimals))
			result.Fee = amount(_formatBigIntToString(transferInfo.Fee, decimals))
		} else {
			result.Amount = amount(transferData.Amount.String())
			result.Fee = amount(transferInfo.Fee.String())
		}
	}
	response.SetResult(result)
}

func (r *BackRpc) _isTokenKnown(symbol string) bool {
	_, found := r.knownTokens[symbol]
	return found
}

type transferAssetsResult struct {
	TxID              string `json:"tx_id"`
	Success           bool   `json:"success"`
	NativeCoin        bool   `json:"nativeCoin,omitempty"`
	SmartContract     bool   `json:"smartContract,omitempty"`
	Symbol            string `json:"symbol,omitempty"`
	From              string `json:"from"`
	To                string `json:"to"`
	Amount            amount `json:"amount"`
	Fee               amount `json:"fee"`
	FeeSymbol         string `json:"feeSymbol,omitempty"`
	Warning           string `json:"warning,omitempty"`
	ChainSpecificData []byte `json:"chainSpecificData,omitempty"`
}

func (s *transferAssetsResult) fill(tx *types.TransferInfo) *transferAssetsResult {
	s.TxID = tx.TxID
	s.Success = tx.Success
	s.NativeCoin = tx.NativeCoin
	s.SmartContract = tx.SmartContract
	s.Symbol = tx.Symbol
	s.From = tx.From
	s.To = tx.To
	s.ChainSpecificData = tx.ChainSpecificData
	return s
}

func (r *BackRpc) rpcProcessTransferGetEstimatedFee(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type transferGetEstimatedFeeRequest struct {
		From           string      `json:"from,omitempty"`
		To             string      `json:"to"`
		Amount         json.Number `json:"amount"`
		Symbol         string      `json:"symbol,omitempty"`
		AmountFormated bool        `json:"amountFormated,omitempty"`
	}
	apiToken, err := ctx.GetApiToken()
	if err == nil {
		if r.debugMode {
			log.Warning("Api token found", apiToken)
		}
	}
	params := new(transferGetEstimatedFeeRequest)
	err = request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	params.From, err = r.addressNormalise(params.From)
	if err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid from address")
		return
	}
	params.To, err = r.addressNormalise(params.To)
	if err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid target address")
		return
	}
	if r.debugMode {
		log.Debug("Transfer assets request")
		log.Dump(params)
	}
	var amountToTransfer *big.Int
	var decimals int
	var ok bool
	if params.AmountFormated {
		if params.Symbol == r.chainClient.GetChainSymbol() {
			decimals = r.chainClient.Decimals()
		} else {
			decimals = r.knownTokens[params.Symbol].Decimals
		}
		amountToTransfer, err = _parseAmountToBigInt(params.Amount.String(), decimals)
		if err != nil {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid amount")
			return
		}
	} else {
		amountToTransfer, ok = new(big.Int).SetString(params.Amount.String(), 10)
		if !ok {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "invalid amount")
			return
		}
	}
	transferData := &struct {
		From   string   `json:"from,omitempty"`
		To     string   `json:"to"`
		Amount *big.Int `json:"amount"`
		Symbol string   `json:"symbol,omitempty"`
	}{
		From:   params.From,
		To:     params.To,
		Amount: amountToTransfer,
		Symbol: params.Symbol,
	}
	var fee *big.Int
	if !r._isTokenKnown(transferData.Symbol) && transferData.Symbol != r.chainClient.GetChainSymbol() {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "unknown asset symbol")
		return
	}
	if transferData.Symbol == r.chainClient.GetChainSymbol() {
		if r.debugMode {
			log.Debug("Transferring native coin request")
		}
		fee, err = r.chainClient.TransferGetEstimatedFee(transferData.From, transferData.To, transferData.Amount)
	} else {
		if r.debugMode {
			log.Debug("Transferring token request")
		}
		fee, err = r.chainClient.TransferTokenGetEstimatedFee(transferData.From, transferData.To, transferData.Amount, transferData.Symbol)
	}
	if err != nil {
		//TODO check is it possible to get error from chain
		if r.debugMode {
			log.Error("Get estimated fee error:", err)
		}
		response.SetError(ERROR_CODE_SERVER_ERROR, err.Error())
		return
	}
	var feeResult json.Number
	if params.AmountFormated {
		feeResult = json.Number(_formatBigIntToString(fee, decimals))
	} else {
		feeResult = json.Number(fee.String())
	}
	response.SetResult(feeResult)
}

func _parseAmountToBigInt(amount string, decimals int) (*big.Int, error) {
	amount = strings.Trim(amount, "\"")
	amount = strings.ReplaceAll(amount, ",", ".")
	amount = strings.ReplaceAll(amount, " ", "")
	amount = strings.ReplaceAll(amount, "\t", "")
	if strings.Contains(amount, ".") {
		parts := strings.Split(amount, ".")
		if len(parts) != 2 {
			return nil, ErrInvalidAmount
		}
		hi := parts[0]
		lo := parts[1]
		if len(lo) < decimals {
			lo = lo + strings.Repeat("0", decimals-len(lo))
		} else if len(lo) > decimals {
			lo = lo[:decimals]
		}
		amount = hi + lo
	} else {
		amount = amount + strings.Repeat("0", decimals)
	}
	amountBig, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return nil, ErrInvalidAmount
	}
	return amountBig, nil
}

func _formatBigIntToString(balance *big.Int, decimals int) string {
	str := balance.String()
	if len(str) > decimals {
		return str[:len(str)-decimals] + "." + str[len(str)-decimals:]
	} else {
		return "0." + strings.Repeat("0", decimals-len(str)) + str
	}
}
