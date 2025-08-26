package database

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Withdraws struct {
	Guid        uuid.UUID `gorm:"primaryKey" json:"guid"`
	BlockHash   string    `json:"block_hash"`
	BlockNumber *big.Int  `gorm:"serializer:u256;check:block_number > 0" json:"block_number"`
	Hash        string    `json:"hash"`
	Fee         *big.Int  `gorm:"serializer:u256" json:"fee"`
	LockTime    *big.Int  `gorm:"serializer:u256" json:"lock_time"`
	Version     string    `json:"version"`
	TxSignHex   string    `json:"tx_sign_hex"`
	Status      uint8     `gorm:"default:0" json:"status"`
	Timestamp   uint64    `json:"timestamp"`
}

type WithdrawsView interface {
	QueryNotifyWithdraws(requestId string) ([]Withdraws, error)

	UnSendWithdrawsList(requestId string) ([]Withdraws, error)
}

type WithdrawsDB interface {
	WithdrawsView

	StoreWithdraws(string, *Withdraws) error
	UpdateWithdrawStatus(requestId string, status TxStatus, withdrawsList []Withdraws) error
}

type withdrawsDB struct {
	gorm *gorm.DB
}

func NewWithdrawsDB(db *gorm.DB) WithdrawsDB {
	return &withdrawsDB{gorm: db}
}

func (db *withdrawsDB) StoreWithdraws(requestId string, withdrawsList *Withdraws) error {
	result := db.gorm.Table("withdraws_" + requestId).Create(&withdrawsList)
	return result.Error
}

func (db *withdrawsDB) UpdateWithdrawStatus(requestId string, status TxStatus, withdrawsList []Withdraws) error {
	if len(withdrawsList) == 0 {
		return nil
	}
	tableName := fmt.Sprintf("withdraws_%s", requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var guids []uuid.UUID
		for _, withdraw := range withdrawsList {
			guids = append(guids, withdraw.Guid)
		}

		result := tx.Table(tableName).
			Where("guid IN ?", guids).
			Where("status = ?", TxStatusWalletDone).
			Update("status", status)

		if result.Error != nil {
			return fmt.Errorf("batch update status failed: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			log.Warn("No withdraws updated",
				"requestId", requestId,
				"expectedCount", len(withdrawsList),
			)
		}

		log.Info("Batch update withdraws status success",
			"requestId", requestId,
			"count", result.RowsAffected,
			"status", status,
		)

		return nil
	})
}

func (db *withdrawsDB) QueryNotifyWithdraws(requestId string) ([]Withdraws, error) {
	var notifyWithdraws []Withdraws
	result := db.gorm.Table("withdraws_"+requestId).
		Where("status = ?", TxStatusWalletDone).
		Find(&notifyWithdraws)

	if result.Error != nil {
		return nil, fmt.Errorf("query notify withdraws failed: %w", result.Error)
	}

	return notifyWithdraws, nil
}

func (db *withdrawsDB) UnSendWithdrawsList(requestId string) ([]Withdraws, error) {
	var withdrawsList []Withdraws
	err := db.gorm.Table("withdraws_"+requestId).
		Where("status = ?", TxStatusSigned).
		Find(&withdrawsList).Error

	if err != nil {
		return nil, fmt.Errorf("query unsend withdraws failed: %w", err)
	}

	return withdrawsList, nil
}
