package worker

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/common/tasks"
	"github.com/dapplink-labs/multichain-sync-btc/config"
	"github.com/dapplink-labs/multichain-sync-btc/database"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient"
)

type Deposit struct {
	BaseSynchronizer

	confirms       uint8
	latestHeader   rpcclient.BlockHeader
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewDeposit(cfg *config.Config, db *database.DB, rpcClient *rpcclient.WalletBtcAccountClient, shutdown context.CancelCauseFunc) (*Deposit, error) {
	dbLatestBlockHeader, err := db.Blocks.LatestBlocks()
	if err != nil {
		log.Error("get latest block from database fail")
		return nil, err
	}
	var fromHeader *rpcclient.BlockHeader

	if dbLatestBlockHeader != nil {
		log.Info("sync bock", "number", dbLatestBlockHeader.Number, "hash", dbLatestBlockHeader.Hash)
		fromHeader = dbLatestBlockHeader
	} else if cfg.ChainNode.StartingHeight > 0 {
		chainLatestBlockHeader, err := rpcClient.GetBlockHeader(big.NewInt(int64(cfg.ChainNode.StartingHeight)))
		if err != nil {
			log.Error("get block from chain account fail", "err", err)
			return nil, err
		}
		fromHeader = chainLatestBlockHeader
	} else {
		chainLatestBlockHeader, err := rpcClient.GetBlockHeader(nil)
		if err != nil {
			log.Error("get block from chain account fail", "err", err)
			return nil, err
		}
		fromHeader = chainLatestBlockHeader
	}

	businessTxChannel := make(chan map[string]*TransactionsChannel)

	baseSyncer := BaseSynchronizer{
		loopInterval:     cfg.ChainNode.SynchronizerInterval,
		headerBufferSize: cfg.ChainNode.BlocksStep,
		businessChannels: businessTxChannel,
		rpcClient:        rpcClient,
		blockBatch:       rpcclient.NewBatchBlock(rpcClient, fromHeader, big.NewInt(int64(cfg.ChainNode.Confirmations))),
		database:         db,
	}

	resCtx, resCancel := context.WithCancel(context.Background())

	return &Deposit{
		BaseSynchronizer: baseSyncer,
		confirms:         uint8(cfg.ChainNode.Confirmations),
		resourceCtx:      resCtx,
		resourceCancel:   resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in deposit: %w", err))
		}},
	}, nil
}

func (deposit *Deposit) Close() error {
	var result error
	if err := deposit.BaseSynchronizer.Close(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to close internal base synchronizer: %w", err))
	}
	deposit.resourceCancel()
	if err := deposit.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await batch handler completion: %w", err))
	}
	return result
}

func (deposit *Deposit) Start() error {
	log.Info("starting deposit...")
	if err := deposit.BaseSynchronizer.Start(); err != nil {
		return fmt.Errorf("failed to start internal Synchronizer: %w", err)
	}
	deposit.tasks.Go(func() error {
		log.Info("handle deposit task start")
		for batch := range deposit.businessChannels {
			log.Info("deposit business channel", "batch length", len(batch))
			if err := deposit.handleBatch(batch); err != nil {
				return fmt.Errorf("failed to handle batch, stopping L2 Synchronizer: %w", err)
			}
		}
		return nil
	})
	return nil
}

func (deposit *Deposit) handleBatch(batch map[string]*TransactionsChannel) error {
	return nil
}
