package database

import (
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Vins struct {
	GUID             uuid.UUID `gorm:"primaryKey" json:"guid"`
	Address          string    `json:"address"`
	TxId             string    `json:"tx_id"`
	Vout             uint8     `json:"vout"`
	Script           string    `json:"script"`
	Witness          string    `json:"witness"`
	Amount           *big.Int  `gorm:"serializer:u256" json:"amount"`
	SpendTxHash      string    `json:"spend_tx_hash"`
	SpendBlockHeight *big.Int  `gorm:"serializer:u256" json:"spend_block_height"`
	IsSpend          bool      `json:"is_spend"`
	Timestamp        uint64    `json:"timestamp"`
}

type VinsView interface {
	QueryVinByTxId(string, string, string) (*Vins, error)
}

type VinsDB interface {
	VinsView

	StoreVins(string, []Vins) error
	UpdateVinsTx(requestId string, txId string, address string, IsSpend bool, spendTxHash string, spendBlockHeight *big.Int) error
}

type vinsDB struct {
	gorm *gorm.DB
}

func NewVinsDB(db *gorm.DB) VinsDB {
	return &vinsDB{gorm: db}
}

func (vin vinsDB) QueryVinByTxId(businessId string, address string, txId string) (*Vins, error) {
	var vinEntry Vins
	err := vin.gorm.Table("vins_"+businessId).Where("tx_id = ? and address = ?", txId, address).Take(&vinEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &vinEntry, nil
}

func (vin vinsDB) StoreVins(businessId string, vins []Vins) error {
	result := vin.gorm.Table("vins_"+businessId).CreateInBatches(&vins, len(vins))
	return result.Error
}

func (vin vinsDB) UpdateVinsTx(requestId string, txId string, address string, IsSpend bool, spendTxHash string, spendBlockHeight *big.Int) error {

	updates := map[string]interface{}{
		"is_spend": false,
	}

	if (spendTxHash != "" && spendBlockHeight != big.NewInt(0)) || IsSpend {
		updates["spend_tx_hash"] = spendTxHash
		updates["spend_block_height"] = spendBlockHeight
		updates["is_spend"] = true
	}

	result := vin.gorm.Table("internals_"+requestId).
		Where("txId = ? and address = ?", txId, address).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
