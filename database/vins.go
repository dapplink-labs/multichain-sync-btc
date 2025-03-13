package database

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Vins struct {
	GUID             uuid.UUID `gorm:"primaryKey" json:"guid"`
	Address          string    `json:"address"`
	TxId             string    `json:"tx_id"`
	Vout             uint8     `json:"vout"`
	Script           string    `json:"script"`
	Witness          string    `json:"witness"`
	Amount           *big.Int  `gorm:"serializer:u256" json:"amount"`
	SpendTxHash      string    `json:"spend_tx_hash"`
	SpendBlockHeight *big.Int  `gorm:"serializer:u256" json:"spend_block_height""`
	IsSpend          bool      `json:"is_spend"`
	Timestamp        uint64    `json:"timestamp"`
}

type VinsView interface {
	QueryVinByTxId(string, string) (*Vins, error)
}

type VinsDB interface {
	VinsView

	StoreVins(string, []Vins) error
}

type vinsDB struct {
	gorm *gorm.DB
}

func NewVinsDB(db *gorm.DB) VinsDB {
	return &vinsDB{gorm: db}
}

func (v vinsDB) QueryVinByTxId(s string, s2 string) (*Vins, error) {
	//TODO implement me
	panic("implement me")
}

func (v vinsDB) StoreVins(s string, vins []Vins) error {
	//TODO implement me
	panic("implement me")
}
