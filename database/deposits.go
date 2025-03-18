package database

import (
	"errors"
	"github.com/ethereum/go-ethereum/log"
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
	UpdateDepositsComfirms(requestId string, blockNumber uint64, confirms uint64) error
}

type depositsDB struct {
	gorm *gorm.DB
}

func NewDepositsDB(db *gorm.DB) DepositsDB {
	return &depositsDB{gorm: db}
}

func (db *depositsDB) StoreDeposits(requestId string, depositList []Deposits) error {
	result := db.gorm.Table("deposits_"+requestId).CreateInBatches(&depositList, len(depositList))
	if result.Error != nil {
		log.Error("create deposit batch fail", "Err", result.Error)
		return result.Error
	}
	return nil
}

// UpdateDepositsComfirms 查询所有还没有过确认位交易，用最新区块减去对应区块更新确认，如果这个大于我们预设的确认位，那么这笔交易可以认为已经入账
func (db *depositsDB) UpdateDepositsComfirms(requestId string, blockNumber uint64, confirms uint64) error {
	var unConfirmDeposits []Deposits
	result := db.gorm.Table("deposits_"+requestId).Where("block_number <= ? and status = ?", blockNumber, 0).Find(&unConfirmDeposits)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		return result.Error
	}
	for _, deposit := range unConfirmDeposits {
		chainConfirm := blockNumber - deposit.BlockNumber.Uint64()
		if chainConfirm >= confirms {
			deposit.Confirms = uint8(confirms)
			deposit.Status = 1 // 已经过了确认位
		} else {
			deposit.Confirms = uint8(chainConfirm)
		}
		err := db.gorm.Table("deposits_" + requestId).Save(&deposit).Error
		if err != nil {
			return err
		}
	}
	return nil
}
