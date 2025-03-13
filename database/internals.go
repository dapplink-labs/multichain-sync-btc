package database

import (
	"gorm.io/gorm"
	"math/big"
)

type Internals struct {
	Guid        string   `gorm:"primaryKey" json:"guid"`
	BlockHash   string   `json:"block_hash"`
	BlockNumber *big.Int `gorm:"serializer:u256;check:block_number > 0" json:"block_number"`
	Hash        string   `json:"hash"`
	FromAddress string   `json:"from_address"`
	ToAddress   string   `json:"to_address"`
	Amount      string   `json:"amount"`
	Fee         *big.Int `gorm:"serializer:u256" json:"fee"`
	LockTime    *big.Int `gorm:"serializer:u256" json:"lock_time"`
	Version     string   `json:"version"`
	TxSignHex   string   `json:"tx_sign_hex"`
	Status      uint8    `gorm:"default:0" json:"status"`
	Timestamp   uint64   `json:"timestamp"`
}

type InternalsView interface {
}

type InternalsDB interface {
	InternalsView

	StoreInternals(string, []Internals) error
}

type internalsDB struct {
	gorm *gorm.DB
}

func NewInternalsDB(db *gorm.DB) InternalsDB {
	return &internalsDB{gorm: db}
}

func (w internalsDB) StoreInternals(businessId string, internals []Internals) error {
	panic("implement me")
}
