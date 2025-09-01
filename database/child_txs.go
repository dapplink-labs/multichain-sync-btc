package database

import (
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
	"math/big"

	"github.com/google/uuid"
)

type ChildTxs struct {
	GUID        uuid.UUID `gorm:"primaryKey" json:"guid"`
	Hash        string    `json:"hash"`
	TxId        string    `json:"tx_id"`
	TxIndex     *big.Int  `gorm:"serializer:u256" json:"tx_index"`
	TxType      string    `json:"tx_type"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	Amount      string    `json:"amount"`
	Timestamp   uint64    `json:"timestamp"`
}

type ChildTxsView interface {
	QueryChildTxnByTxId(string, string) ([]ChildTxs, error)
}

type ChildTxsDB interface {
	ChildTxsView

	StoreChildTxs(string, []ChildTxs) error
}

type childTxsDB struct {
	gorm *gorm.DB
}

func NewChildTxsDB(db *gorm.DB) ChildTxsDB {
	return &childTxsDB{gorm: db}
}

func (c childTxsDB) StoreChildTxs(businessId string, txs []ChildTxs) error {
	err := c.gorm.Table("child_txs_"+businessId).CreateInBatches(txs, len(txs)).Error
	if err != nil {
		log.Error("create in batches fail", "err", err)
		return err
	}
	return nil
}

func (c childTxsDB) QueryChildTxnByTxId(businessId string, txId string) ([]ChildTxs, error) {
	var childTxList []ChildTxs
	err := c.gorm.Table("child_txs_"+businessId).Where("tx_id = ?", txId).Find(&childTxList).Error
	if err != nil {
		log.Error("query child txn fail", "err", err)
		return nil, err
	}
	return childTxList, nil
}
