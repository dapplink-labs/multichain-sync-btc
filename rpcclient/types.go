package rpcclient

import (
	"math/big"
)

type BlockHeader struct {
	Hash      string
	PrevHash  string
	Number    *big.Int
	Timestamp uint64
}
