package ethclient

import (
	"fmt"
	"math/big"

	"github.com/ITProLabDev/ethbacknode/clients/urpc"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/tools/log"
)

// EIP-1559 pricing, and the choice of which envelope to sign.
//
// A legacy transaction names one price and pays it in full. A dynamic-fee
// transaction names a ceiling and a tip, and the chain charges the base fee
// plus the tip, refunding the rest -- which is cheaper for the sender on any
// chain that has a base fee. Which envelope to use is therefore decided from
// the chain, not configured: the node knows whether it has a base fee, and a
// wrong guess either gets the transaction rejected or overcharges the sender.

// ethMaxPriorityFeePerGas is the node's own suggestion for a tip. Not every
// node implements it -- it is a geth extension that became conventional --
// so a refusal is handled rather than propagated.
const ethMaxPriorityFeePerGas = "eth_maxPriorityFeePerGas"

// defaultPriorityFeeWei is the tip offered when the node will not suggest
// one. One gwei: a guess, but one that produces a mineable transaction
// rather than one offering nothing.
var defaultPriorityFeeWei = big.NewInt(1_000_000_000)

// feeCapMultiplierPercent is how much of the base fee the cap allows for,
// before the tip is added. 200 is twice the base fee -- headroom for the
// base fee rising before the transaction is mined, which it can do by up to
// 12.5% a block.
const feeCapMultiplierPercent = 200

// txFees is what a transaction is priced at, whichever envelope it uses.
type txFees struct {
	dynamic bool

	// gasPrice is set for a legacy transaction and nil otherwise.
	gasPrice *big.Int

	// maxFeePerGas and maxPriorityFeePerGas are set for a dynamic-fee
	// transaction and nil otherwise.
	maxFeePerGas, maxPriorityFeePerGas *big.Int
}

// cap returns the most the transaction can cost per unit of gas, whichever
// envelope it is. That is the figure a balance check has to use.
func (f txFees) cap() *big.Int {
	if f.dynamic {
		return f.maxFeePerGas
	}
	return f.gasPrice
}

// GetBlockByTag returns the block named by a tag ("latest", "pending",
// "earliest"), unlike GetBlockByNumber which always names a specific height.
func (c *Client) GetBlockByTag(tag string, fullTransactions bool) (*Block, error) {
	req := urpc.NewRequest(ethGetBlockByNumber)
	req.AddParams(tag, fullTransactions)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		return nil, err
	}
	block := new(Block)
	block.FullTransactions = fullTransactions
	if err := result.ParseResult(block); err != nil {
		return nil, err
	}
	return block, nil
}

// BaseFee returns the latest block's base fee, or nil when the chain has
// none -- a pre-London chain, which will only ever take a legacy
// transaction.
func (c *Client) BaseFee() (*big.Int, error) {
	block, err := c.GetBlockByTag(tagBlockLatest, false)
	if err != nil {
		return nil, err
	}
	return block.BaseFeePerGas, nil
}

// SuggestPriorityFee returns the tip to offer: the node's own suggestion via
// eth_maxPriorityFeePerGas, or defaultPriorityFeeWei when the node does not
// implement the call or answers something this cannot parse.
//
// Any failure here falls back rather than propagating, unlike BaseFee and
// GasPrice above it in resolveFees: a node that is actually unreachable
// already fails there first, so a failure reaching this call is ordinarily
// the node simply not implementing an optional method.
func (c *Client) SuggestPriorityFee() (*big.Int, error) {
	req := urpc.NewRequest(ethMaxPriorityFeePerGas)
	result, err := c.rpcClient.Call(req)
	if err != nil {
		log.Debug("eth_maxPriorityFeePerGas unavailable, offering the default tip:", defaultPriorityFeeWei, "-", err)
		return new(big.Int).Set(defaultPriorityFeeWei), nil
	}
	var raw string
	if err := result.ParseResult(&raw); err != nil {
		log.Debug("eth_maxPriorityFeePerGas answer unparseable, offering the default tip:", defaultPriorityFeeWei, "-", err)
		return new(big.Int).Set(defaultPriorityFeeWei), nil
	}
	value, err := hexnum.ParseBigInt(raw)
	if err != nil {
		log.Debug("eth_maxPriorityFeePerGas returned", raw, ", offering the default tip:", defaultPriorityFeeWei, "-", err)
		return new(big.Int).Set(defaultPriorityFeeWei), nil
	}
	return value, nil
}

// checkFeeCeiling refuses a price above the configured ceiling
// (Config.FeeCeilingGwei, 0 = no ceiling).
//
// Refused rather than clamped, which matters more than it looks: clamping
// signs a transaction the chain will not mine at what it is charging, which
// consumes a nonce and blocks every later transaction from that address
// until somebody notices. Refusing takes no nonce -- fees are resolved
// before the nonce for exactly this reason.
func (c *Client) checkFeeCeiling(what string, price *big.Int) error {
	if c.config == nil || c.config.FeeCeilingGwei <= 0 || price == nil {
		return nil
	}
	ceiling := new(big.Int).Mul(big.NewInt(c.config.FeeCeilingGwei), big.NewInt(1_000_000_000))
	if price.Cmp(ceiling) <= 0 {
		return nil
	}
	return fmt.Errorf("%w: %s would be %s wei per gas and the ceiling is %s",
		ErrFeeCeilingExceeded, what, price, ceiling)
}

// resolveFees decides the envelope and prices it: dynamic-fee on a chain
// that reports a base fee, legacy on one that does not.
func (c *Client) resolveFees() (txFees, error) {
	baseFee, err := c.BaseFee()
	if err != nil {
		return txFees{}, err
	}
	if baseFee == nil {
		price, err := c.GasPrice()
		if err != nil {
			return txFees{}, err
		}
		if err := c.checkFeeCeiling("the gas price", price); err != nil {
			return txFees{}, err
		}
		return txFees{gasPrice: price}, nil
	}

	tip, err := c.SuggestPriorityFee()
	if err != nil {
		return txFees{}, err
	}
	cap := new(big.Int).Mul(baseFee, big.NewInt(feeCapMultiplierPercent))
	cap.Div(cap, big.NewInt(100))
	cap.Add(cap, tip)
	if err := c.checkFeeCeiling("the fee cap", cap); err != nil {
		return txFees{}, err
	}
	return txFees{dynamic: true, maxFeePerGas: cap, maxPriorityFeePerGas: tip}, nil
}

// resolveTransferFees prices a plain value transfer: the gas from the node's
// estimate and the price from whichever envelope the chain takes
// (resolveFees). fee is the most the transaction can cost -- the cap times
// the gas -- which is what a balance check must use, on either envelope.
func (c *Client) resolveTransferFees(from, to string, amountForGasEstimate *big.Int) (fees txFees, gas int64, fee *big.Int, err error) {
	gas, err = c.GetEstimatedGas(from, to, "", amountForGasEstimate)
	if err != nil {
		return txFees{}, 0, nil, err
	}
	log.Warning("Estimated Gas:", gas)
	fees, err = c.resolveFees()
	if err != nil {
		return txFees{}, 0, nil, err
	}
	fee = new(big.Int).Mul(fees.cap(), big.NewInt(gas))
	return fees, gas, fee, nil
}
