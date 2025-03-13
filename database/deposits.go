package database

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Deposits struct {
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
	Confirms    uint8    `json:"confirms"`
	Status      uint8    `json:"status"`
	Timestamp   uint64   `json:"timestamp"`
}

type DepositsView interface {
}

type DepositsDB interface {
	DepositsView

	StoreDeposits(string, []Deposits) error
}

type depositsDB struct {
	gorm *gorm.DB
}

func NewDepositsDB(db *gorm.DB) DepositsDB {
	return &depositsDB{gorm: db}
}

func (v depositsDB) StoreDeposits(businessId string, deposits []Deposits) error {
	panic("implement me")
}
