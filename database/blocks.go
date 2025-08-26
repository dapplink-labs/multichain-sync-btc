package database

import (
	"errors"
	"gorm.io/gorm"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/syncclient"
)

type Blocks struct {
	Hash      string `gorm:"primaryKey"`
	PrevHash  string
	Number    *big.Int `gorm:"serializer:u256"`
	Timestamp uint64
}

func BlockHeaderFromHeader(header *types.Header) syncclient.BlockHeader {
	return syncclient.BlockHeader{}
}

type BlocksView interface {
	LatestBlocks() (*syncclient.BlockHeader, error)
}

type BlocksDB interface {
	BlocksView

	StoreBlockss([]Blocks) error
}

type blocksDB struct {
	gorm *gorm.DB
}

func NewBlocksDB(db *gorm.DB) BlocksDB {
	return &blocksDB{gorm: db}
}

func (db *blocksDB) StoreBlockss(headers []Blocks) error {
	result := db.gorm.CreateInBatches(&headers, len(headers))
	return result.Error
}

func (db *blocksDB) LatestBlocks() (*syncclient.BlockHeader, error) {
	var header Blocks
	result := db.gorm.Order("number DESC").Take(&header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return (*syncclient.BlockHeader)(&header), nil
}
