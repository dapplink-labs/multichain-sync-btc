package database

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Transactions struct {
	GUID        uuid.UUID `gorm:"primaryKey" json:"guid"`
	BlockHash   string
	BlockNumber *big.Int `gorm:"serializer:u256"`
	Hash        string   `json:"hash"`
	FromAddress string   `json:"from_address"`
	ToAddress   string   `json:"to_address"`
	Amount      string   `json:"amount"`
	Fee         *big.Int `gorm:"serializer:u256"`
	LockTime    *big.Int `gorm:"serializer:u256"`
	Version     string   `json:"version"`
	Status      uint8    `json:"status"`
	Timestamp   uint64   `json:"timestamp"`
}

type TransactionsView interface {
}

type TransactionsDB interface {
	TransactionsView

	StoreTransactions(string, []Transactions) error
}

type tansactionsDB struct {
	gorm *gorm.DB
}

func NewTransactionsDB(db *gorm.DB) TransactionsDB {
	return &tansactionsDB{gorm: db}
}

func (v tansactionsDB) StoreTransactions(businessId string, tansactions []Transactions) error {
	panic("implement me")
}
