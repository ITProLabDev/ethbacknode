package ethclient

import (
	"fmt"
	"math/big"
)

func WeiToEtherFloat(wei *big.Int) *big.Float {
	weiFloat := new(big.Float).SetInt(wei)
	ether := new(big.Float).Quo(weiFloat, big.NewFloat(1e18))
	return ether
}

func WeiToEtherString(wei *big.Int) string {
	w2eK := 1000000000000000000
	hi := new(big.Int).Div(wei, big.NewInt(int64(w2eK)))
	lo := new(big.Int).Mod(wei, big.NewInt(int64(w2eK)))
	return fmt.Sprintf("%s.%018s", hi.String(), lo.String())
}
