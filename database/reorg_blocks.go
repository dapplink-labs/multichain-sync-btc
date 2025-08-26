package database

import (
	"errors"
	"math/big"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/syncclient"
)

type ReorgBlocks struct {
	Hash      string `gorm:"primaryKey"`
	PrevHash  string
	Number    *big.Int `gorm:"serializer:u256"`
	Timestamp uint64
}

func ReorgBlockHeaderFromHeader(header *types.Header) syncclient.BlockHeader {
	return syncclient.BlockHeader{
		Hash:      header.Hash().String(),
		PrevHash:  header.ParentHash.String(),
		Number:    header.Number,
		Timestamp: header.Time,
	}
}

type ReorgBlocksView interface {
	LatestReorgBlocks() (*syncclient.BlockHeader, error)
}

type ReorgBlocksDB interface {
	ReorgBlocksView

	StoreReorgBlocks([]ReorgBlocks) error
}

type reorgBlocksDB struct {
	gorm *gorm.DB
}

func NewReorgBlocksDB(db *gorm.DB) ReorgBlocksDB {
	return &reorgBlocksDB{gorm: db}
}

func (db *reorgBlocksDB) StoreReorgBlocks(headers []ReorgBlocks) error {
	result := db.gorm.CreateInBatches(&headers, len(headers))
	return result.Error
}

func (db *reorgBlocksDB) LatestReorgBlocks() (*syncclient.BlockHeader, error) {
	var header ReorgBlocks
	result := db.gorm.Order("number DESC").Take(&header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return (*syncclient.BlockHeader)(&header), nil
}
