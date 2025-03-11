package database

import (
	"math/big"
)

type TokenBalance struct {
	FromAddress  string   `json:"from_address"`
	ToAddress    string   `json:"to_address"`
	TokenAddress string   `json:"to_ken_address"`
	Balance      *big.Int `json:"balance"`
	TxType       string   `json:"tx_type"` // deposit:充值；withdraw:提现；collection:归集；hot2cold:热转冷；cold2hot:冷转热
}
