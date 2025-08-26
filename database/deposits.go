package database

import (
	"errors"
	"gorm.io/gorm"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

type Deposits struct {
	GUID        uuid.UUID `gorm:"primaryKey" json:"guid"`
	BlockHash   string
	BlockNumber *big.Int `gorm:"serializer:u256"`
	Hash        string   `json:"hash"`
	Fee         *big.Int `gorm:"serializer:u256"`
	LockTime    *big.Int `gorm:"serializer:u256"`
	Version     string   `json:"version"`
	Confirms    uint8    `json:"confirms"`
	Status      uint8    `json:"status"`
	Timestamp   uint64   `json:"timestamp"`
}

type DepositsView interface {
	QueryNotifyDeposits(string) ([]Deposits, error)
}

type DepositsDB interface {
	DepositsView

	StoreDeposits(string, []Deposits) error
	UpdateDepositsComfirms(requestId string, blockNumber uint64, confirms uint64) error
	UpdateDepositsNotifyStatus(requestId string, status uint8, depositList []Deposits) error
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

func (db *depositsDB) QueryNotifyDeposits(requestId string) ([]Deposits, error) {
	var notifyDeposits []Deposits
	result := db.gorm.Table("deposits_"+requestId).Where("status = ? or status = ?", 0, 1).Find(notifyDeposits)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, result.Error
	}
	return notifyDeposits, nil
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

func (db *depositsDB) UpdateDepositsNotifyStatus(requestId string, status uint8, depositList []Deposits) error {
	for i := 0; i < len(depositList); i++ {
		var depositSingle = Deposits{}
		result := db.gorm.Table("deposits_" + requestId).Where(&Transactions{Hash: depositList[i].Hash}).Take(&depositSingle)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil
			}
			return result.Error
		}
		depositSingle.Status = status
		err := db.gorm.Table("transactions_" + requestId).Save(&depositSingle).Error
		if err != nil {
			return err
		}
	}
	return nil
}
