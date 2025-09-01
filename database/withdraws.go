package database

import (
	"fmt"

	"gorm.io/gorm"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
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
	Status      TxStatus  `json:"status"`
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
	UpdateWithdrawByGuuid(requestId string, transactionId string, txSignedHex string) error
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
			Where("status = ?", TxStatusWithdrawed).
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

func (db *withdrawsDB) UpdateWithdrawByGuuid(requestId string, transactionId string, txSignedHex string) error {
	tableName := fmt.Sprintf("withdraws_%s", requestId)
	var withdrawItem Withdraws
	result := db.gorm.Table(tableName).Where("guid = ?", transactionId).Take(&withdrawItem)
	if result.Error != nil {
		log.Error("query fail", "err", result.Error)
	}

	withdrawItem.TxSignHex = txSignedHex
	withdrawItem.Status = TxStatusUnSent

	err := db.gorm.Table(tableName).Save(withdrawItem).Error
	if err != nil {
		log.Error("update tx fail", "err", err)
		return err
	}
	return nil
}

func (db *withdrawsDB) QueryNotifyWithdraws(requestId string) ([]Withdraws, error) {
	var notifyWithdraws []Withdraws
	result := db.gorm.Table("withdraws_"+requestId).
		Where("status = ?", TxStatusWithdrawed).
		Find(&notifyWithdraws)

	if result.Error != nil {
		return nil, fmt.Errorf("query notify withdraws failed: %w", result.Error)
	}

	return notifyWithdraws, nil
}

func (db *withdrawsDB) UnSendWithdrawsList(requestId string) ([]Withdraws, error) {
	var withdrawsList []Withdraws
	err := db.gorm.Table("withdraws_"+requestId).
		Where("status = ?", TxStatusUnSent).
		Find(&withdrawsList).Error

	if err != nil {
		return nil, fmt.Errorf("query unsend withdraws failed: %w", err)
	}

	return withdrawsList, nil
}
