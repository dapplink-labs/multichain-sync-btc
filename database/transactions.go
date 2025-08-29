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
	Fee         *big.Int `gorm:"serializer:u256"`
	LockTime    *big.Int `gorm:"serializer:u256"`
	TxType      string   `json:"tx_type"`
	Version     string   `json:"version"`
	Status      TxStatus `json:"status"`
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

func (db *tansactionsDB) StoreTransactions(requestId string, transactionsList []Transactions) error {
	result := db.gorm.Table("transactions_"+requestId).CreateInBatches(&transactionsList, len(transactionsList))
	return result.Error
}
