package bip44

var coinTypes = make(map[string]uint32)

func init() {
	coinTypes[`Bitcoin`] = 0x80000000
	coinTypes[`Testnet`] = 0x80000001
	coinTypes[`Litecoin`] = 0x80000002
	coinTypes[`Dogecoin`] = 0x80000003
	coinTypes[`Reddcoin`] = 0x80000004
	coinTypes[`Dash`] = 0x80000005
	coinTypes[`Peercoin`] = 0x80000006
	coinTypes[`Namecoin`] = 0x80000007
	coinTypes[`Feathercoin`] = 0x80000008
	coinTypes[`Counterparty`] = 0x80000009
	coinTypes[`Blackcoin`] = 0x8000000a
	coinTypes[`NuShares`] = 0x8000000b
	coinTypes[`NuBits`] = 0x8000000c
	coinTypes[`Mazacoin`] = 0x8000000d
	coinTypes[`Viacoin`] = 0x8000000e
	coinTypes[`ClearingHouse`] = 0x8000000f
	coinTypes[`Rubycoin`] = 0x80000010
	coinTypes[`Groestlcoin`] = 0x80000011
	coinTypes[`Digitalcoin`] = 0x80000012
	coinTypes[`Cannacoin`] = 0x80000013
	coinTypes[`DigiByte`] = 0x80000014
	coinTypes[`OpenAssets`] = 0x80000015
	coinTypes[`Monacoin`] = 0x80000016
	coinTypes[`Clams`] = 0x80000017
	coinTypes[`Primecoin`] = 0x80000018
	coinTypes[`Neoscoin`] = 0x80000019
	coinTypes[`Jumbucks`] = 0x8000001a
	coinTypes[`ziftrCOIN`] = 0x8000001b
	coinTypes[`Vertcoin`] = 0x8000001c
	coinTypes[`NXT`] = 0x8000001d
	coinTypes[`Burst`] = 0x8000001e
	coinTypes[`MonetaryUnit`] = 0x8000001f
	coinTypes[`Zoom`] = 0x80000020
	coinTypes[`Vpncoin`] = 0x80000021
	coinTypes[`CanadaeCoin`] = 0x80000022
	coinTypes[`ShadowCash`] = 0x80000023
	coinTypes[`ParkByte`] = 0x80000024
	coinTypes[`Pandacoin`] = 0x80000025
	coinTypes[`StartCOIN`] = 0x80000026
	coinTypes[`MOIN`] = 0x80000027
	coinTypes[`Argentum`] = 0x8000002D
	coinTypes[`GlobalCurrencyReserve`] = 0x80000031
	coinTypes[`Novacoin`] = 0x80000032
	coinTypes[`Asiacoin`] = 0x80000033
	coinTypes[`Bitcoindark`] = 0x80000034
	coinTypes[`Dopecoin`] = 0x80000035
	coinTypes[`Templecoin`] = 0x80000036
	coinTypes[`AIB`] = 0x80000037
	coinTypes[`EDRCoin`] = 0x80000038
	coinTypes[`Syscoin`] = 0x80000039
	coinTypes[`Solarcoin`] = 0x8000003a
	coinTypes[`Smileycoin`] = 0x8000003b
	coinTypes[`Ether`] = 0x8000003c
	coinTypes[`EtherClassic`] = 0x8000003d
	coinTypes[`OpenChain`] = 0x80000040
	coinTypes[`OKCash`] = 0x80000045
	coinTypes[`DogecoinDark`] = 0x8000004d
	coinTypes[`ElectronicGulden`] = 0x8000004e
	coinTypes[`ClubCoin`] = 0x8000004f
	coinTypes[`RichCoin`] = 0x80000050
	coinTypes[`Potcoin`] = 0x80000051
	coinTypes[`Quarkcoin`] = 0x80000052
	coinTypes[`Terracoin`] = 0x80000053
	coinTypes[`Gridcoin`] = 0x80000054
	coinTypes[`Auroracoin`] = 0x80000055
	coinTypes[`IXCoin`] = 0x80000056
	coinTypes[`Gulden`] = 0x80000057
	coinTypes[`BitBean`] = 0x80000058
	coinTypes[`Bata`] = 0x80000059
	coinTypes[`Myriadcoin`] = 0x8000005a
	coinTypes[`BitSend`] = 0x8000005b
	coinTypes[`Unobtanium`] = 0x8000005c
	coinTypes[`MasterTrader`] = 0x8000005d
	coinTypes[`GoldBlocks`] = 0x8000005e
	coinTypes[`Saham`] = 0x8000005f
	coinTypes[`Chronos`] = 0x80000060
	coinTypes[`Ubiquoin`] = 0x80000061
	coinTypes[`Evotion`] = 0x80000062
	coinTypes[`SaveTheOcean`] = 0x80000063
	coinTypes[`BigUp`] = 0x80000064
	coinTypes[`GameCredits`] = 0x80000065
	coinTypes[`Dollarcoins`] = 0x80000066
	coinTypes[`Zayedcoin`] = 0x80000067
	coinTypes[`Dubaicoin`] = 0x80000068
	coinTypes[`Stratis`] = 0x80000069
	coinTypes[`Shilling`] = 0x8000006a
	coinTypes[`PiggyCoin`] = 0x80000076
	coinTypes[`Monero`] = 0x80000080
	coinTypes[`NavCoin`] = 0x80000082
	coinTypes[`Zcash`] = 0x80000085
	coinTypes[`Lisk`] = 0x80000086

}

func CoinType(coin string) uint32 {
	if coinType, ok := coinTypes[coin]; ok {
		return coinType
	}
	return TypeEther
}
