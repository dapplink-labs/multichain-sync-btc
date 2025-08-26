package database

import (
	"gorm.io/gorm"
	"math/big"

	"github.com/google/uuid"
)

type ChildTxs struct {
	GUID        uuid.UUID `gorm:"primaryKey" json:"guid"`
	Hash        string    `json:"hash"`
	TxIndex     *big.Int  `gorm:"serializer:u256" json:"tx_index"`
	TxType      string    `json:"tx_type"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	Amount      string    `json:"amount"`
	Timestamp   uint64    `json:"timestamp"`
}

type ChildTxsView interface {
}

type ChildTxsDB interface {
	ChildTxsView
}

type childTxsDB struct {
	gorm *gorm.DB
}

func NewChildTxsDB(db *gorm.DB) ChildTxsDB {
	return &childTxsDB{gorm: db}
}
