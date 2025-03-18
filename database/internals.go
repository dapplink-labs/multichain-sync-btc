package database

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Internals struct {
	Guid        uuid.UUID `gorm:"primaryKey" json:"guid"`
	BlockHash   string    `json:"block_hash"`
	BlockNumber *big.Int  `gorm:"serializer:u256;check:block_number > 0" json:"block_number"`
	Hash        string    `json:"hash"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	Amount      string    `json:"amount"`
	Fee         *big.Int  `gorm:"serializer:u256" json:"fee"`
	LockTime    *big.Int  `gorm:"serializer:u256" json:"lock_time"`
	Version     string    `json:"version"`
	TxType      string    `json:"tx_type"`
	TxSignHex   string    `json:"tx_sign_hex"`
	Status      uint8     `gorm:"default:0" json:"status"`
	Timestamp   uint64    `json:"timestamp"`
}

type InternalsView interface {
	UnSendInternalsList(requestId string) ([]Internals, error)
}

type InternalsDB interface {
	InternalsView

	StoreInternal(string, *Internals) error
	UpdateInternalTx(requestId string, transactionId string, signedTx string, status TxStatus) error
	UpdateInternalStatus(requestId string, status TxStatus, internalsList []Internals) error
}

type internalsDB struct {
	gorm *gorm.DB
}

func NewInternalsDB(db *gorm.DB) InternalsDB {
	return &internalsDB{gorm: db}
}

func (db *internalsDB) StoreInternal(requestId string, internals *Internals) error {
	return db.gorm.Table("internals_" + requestId).Create(internals).Error
}

func (db *internalsDB) UpdateInternalTx(requestId string, transactionId string, signedTx string, status TxStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	result := db.gorm.Table("internals_"+requestId).
		Where("guid = ?", transactionId).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (db *internalsDB) UpdateInternalStatus(requestId string, status TxStatus, internalsList []Internals) error {
	if len(internalsList) == 0 {
		return nil
	}
	tableName := fmt.Sprintf("internals_%s", requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var guids []uuid.UUID
		for _, internal := range internalsList {
			guids = append(guids, internal.Guid)
		}

		result := tx.Table(tableName).
			Where("guid IN ?", guids).
			Where("status = ?", TxStatusWalletDone).
			Update("status", status)

		if result.Error != nil {
			return fmt.Errorf("batch update status failed: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			log.Warn("No internals updated",
				"requestId", requestId,
				"expectedCount", len(internalsList),
			)
		}

		log.Info("Batch update internals status success",
			"requestId", requestId,
			"count", result.RowsAffected,
			"status", status,
		)

		return nil
	})
}

func (db *internalsDB) UnSendInternalsList(requestId string) ([]Internals, error) {
	var internalsList []Internals
	err := db.gorm.Table("internals_"+requestId).
		Where("status = ?", TxStatusSigned).
		Find(&internalsList).Error
	if err != nil {
		return nil, err
	}
	return internalsList, nil
}
