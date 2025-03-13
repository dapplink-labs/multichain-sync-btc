package database

import (
	"gorm.io/gorm"
	"math/big"
)

type Withdraws struct {
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

type WithdrawsView interface {
}

type WithdrawsDB interface {
	WithdrawsView

	StoreWithdraws(string, []Withdraws) error
}

type withdrawsDB struct {
	gorm *gorm.DB
}

func NewWithdrawsDB(db *gorm.DB) WithdrawsDB {
	return &withdrawsDB{gorm: db}
}

func (w withdrawsDB) StoreWithdraws(businessId string, withdraws []Withdraws) error {
	panic("implement me")
}
