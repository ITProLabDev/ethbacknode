package endpoint

import (
	"bytes"
	"errors"

	"github.com/ITProLabDev/ethbacknode/address"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/subscriptions"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

func (r *BackRpc) rpcProcessAddressSubscribe(ctx RequestContext, request RpcRequest, response RpcResponse) {
	var newAddress string
	var privateKey []byte
	type addressSubscribeRequest struct {
		Address    string                  `json:"address"`
		PrivateKey string                  `json:"privateKey"`
		Mnemonic   []string                `json:"mnemonic,omitempty"` //TODO PROCESS VALIDATE
		ServiceId  subscriptions.ServiceId `json:"serviceId"`
		UserId     int64                   `json:"userId"`
		InvoiceId  int64                   `json:"invoiceId"`
		WatchOnly  bool                    `json:"watchOnly"`
	}
	type addressSubscribeResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	params := new(addressSubscribeRequest)
	err := request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}

	if params.ServiceId == 0 {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "Invalid service id")
		return
	}
	params.Address, err = r.addressNormalise(params.Address)
	if err != nil {
		response.SetError(ERROR_CODE_INVALID_REQUEST, err.Error())
		return
	}
	if params.Address != "" && !r.addressCodec.IsValid(params.Address) {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "Invalid address")
		return
	} else if params.PrivateKey != "" {
		privateKey, err = hexnum.ParseHexBytes(params.PrivateKey)
		if err != nil {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "Invalid private key")
			return
		}
		addressStr, _, err := r.chainClient.GetAddressCodec().PrivateKeyToAddress(privateKey)
		if err != nil {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "Invalid private key")
			return
		}
		if params.Address != "" && addressStr != params.Address {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "Address and private key mismatch")
			return
		}
		params.Address = addressStr
	} else if params.Address == "" && params.PrivateKey == "" {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "Address or private key required")
		return
	}
	subscription, err := r.subscriptions.SubscriptionGet(params.ServiceId)
	if err != nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, err.Error())
		return
	}
	if subscription.Internal {
		response.SetError(ERROR_CODE_SERVER_ERROR, "unknown serviceId")
		return
	}
	if subscription.ApiToken != "" || subscription.ApiKey != "" {
		//TODO: check authorization
		log.Warning("TODO: Authorization needed")
	}
	newAddress = params.Address
	if params.PrivateKey == "" {
		params.WatchOnly = true
	}
	if params.PrivateKey != "" {
		privateKey, err = hexnum.ParseHexBytes(params.PrivateKey)
		if err != nil {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "Invalid private key format")
			return
		}
	}
	if len(params.Mnemonic) > 0 {
		ar, err := r.addressPool.RecoverBit44Address(params.Mnemonic)
		if err != nil {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "Invalid mnemonic")
			return
		}
		if newAddress != "" && newAddress != ar.Address {
			response.SetError(ERROR_CODE_INVALID_REQUEST, "Address and mnemonic mismatch")
			return
		}
		if privateKey != nil {
			if bytes.Compare(privateKey, ar.PrivateKey) != 0 {
				response.SetError(ERROR_CODE_INVALID_REQUEST, "Private key and mnemonic mismatch")
				return
			}
		}
	}
	_, err = r.addressPool.AddAddressFill(newAddress, func(a *address.Address) {
		a.PrivateKey = privateKey
		a.ServiceId = int(params.ServiceId)
		a.UserId = params.UserId
		a.InvoiceId = params.InvoiceId
		a.WatchOnly = params.WatchOnly
		a.Subscribed = true
		if len(params.Mnemonic) > 0 {
			a.Bip39Support = true
			a.Bip39Mnemonic = params.Mnemonic
		}
	})
	if errors.Is(err, address.ErrAddressExists) {
		response.SetResult(addressSubscribeResponse{Success: true, Message: "Address already known"})
		return
	}
	if err != nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, err.Error())
		return
	}
	response.SetResult(&addressSubscribeResponse{Success: true})
}

func (r *BackRpc) rpcProcessAddressGetNew(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type addressGetNewRequest struct {
		ServiceId subscriptions.ServiceId `json:"serviceId"`
		UserId    int64                   `json:"userId"`
		InvoiceId int64                   `json:"invoiceId"`
		WatchOnly bool                    `json:"watchOnly"`
		FullInfo  bool                    `json:"fullInfo"`
	}
	type addressGetNewResponse struct {
		Success       bool     `json:"success"`
		Address       string   `json:"address"`
		PrivateKey    string   `json:"privateKey,omitempty"`
		UserId        int64    `json:"userId,omitempty"`
		InvoiceId     int64    `json:"invoiceId,omitempty"`
		WatchOnly     bool     `json:"watchOnly,omitempty"`
		Bip39Support  bool     `json:"bip39Support,omitempty"`
		Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
	}
	params := new(addressGetNewRequest)
	err := request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	if params.ServiceId == 0 {
		response.SetError(ERROR_CODE_INVALID_REQUEST, "Invalid service id")
		return
	}
	subscription, err := r.subscriptions.SubscriptionGet(params.ServiceId)
	if err != nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, err.Error())
		return
	}
	if subscription.Internal {
		response.SetError(ERROR_CODE_SERVER_ERROR, "unknown serviceId")
		return
	}
	if subscription.ApiToken != "" || subscription.ApiKey != "" {
		//TODO: check authorization
		log.Warning("TODO: Authorization needed")
	}
	newAddress, err := r.addressPool.GetFreeAddressAndSubscribe(int(params.ServiceId), params.UserId, params.InvoiceId, params.WatchOnly)
	if err != nil {
		response.SetError(ERROR_CODE_SERVER_ERROR, err.Error())
		return
	}
	newAddressResponse := &addressGetNewResponse{
		Success: true,
		Address: newAddress.Address,
	}
	if params.FullInfo {
		newAddressResponse.PrivateKey = hexnum.BytesToHex(newAddress.PrivateKey)
		newAddressResponse.UserId = newAddress.UserId
		newAddressResponse.InvoiceId = newAddress.InvoiceId
		newAddressResponse.WatchOnly = newAddress.WatchOnly
		if newAddress.Bip39Support {
			newAddressResponse.Bip39Support = true
			newAddressResponse.Bip39Mnemonic = newAddress.Bip39Mnemonic
		}
	}
	response.SetResult(newAddressResponse)
}

func (r *BackRpc) rpcProcessAddressRecover(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type addressRecoverRequest struct {
		Mnemonic []string `json:"mnemonic"` //TODO PROCESS VALIDATE
	}
	type addressRecoverResponse struct {
		Success       bool     `json:"success"`
		Address       string   `json:"address,omitempty"`
		PrivateKey    string   `json:"privateKey,omitempty"`
		Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
		Error         string   `json:"error,omitempty"`
	}
	params := new(addressRecoverRequest)
	err := request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	addressRecord, err := r.addressPool.RecoverBit44Address(params.Mnemonic)
	if err != nil {
		result := &addressRecoverResponse{
			Success: false,
			Error:   err.Error(),
		}
		response.SetResult(result)
		return
	}
	result := &addressRecoverResponse{
		Success:       true,
		Address:       addressRecord.Address,
		PrivateKey:    hexnum.BytesToHex(addressRecord.PrivateKey),
		Bip39Mnemonic: addressRecord.Bip39Mnemonic,
	}
	response.SetResult(result)
}

func (r *BackRpc) rpcProcessAddressGenerate(ctx RequestContext, request RpcRequest, response RpcResponse) {
	type addressGenerateRequest struct {
		MnemonicLen int `json:"mnemonicLen"` //TODO PROCESS VALIDATE
	}
	type addressRecoverResponse struct {
		Success       bool     `json:"success"`
		Address       string   `json:"address,omitempty"`
		PrivateKey    string   `json:"privateKey,omitempty"`
		Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
		Error         string   `json:"error,omitempty"`
	}
	params := &addressGenerateRequest{
		MnemonicLen: 12,
	}
	err := request.ParseParams(params)
	if err != nil {
		response.SetError(ERROR_CODE_PARSE_ERROR, ERROR_MESSAGE_PARSE_ERROR)
		return
	}
	addressRecord, err := r.addressPool.GenerateBit44AddressWithLen(params.MnemonicLen)
	if err != nil {
		result := &addressRecoverResponse{
			Success: false,
			Error:   err.Error(),
		}
		response.SetResult(result)
		return
	}
	result := &addressRecoverResponse{
		Success:       true,
		Address:       addressRecord.Address,
		PrivateKey:    hexnum.BytesToHex(addressRecord.PrivateKey),
		Bip39Mnemonic: addressRecord.Bip39Mnemonic,
	}
	response.SetResult(result)
}
