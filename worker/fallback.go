package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/common/tasks"
	"github.com/dapplink-labs/multichain-sync-btc/config"
	"github.com/dapplink-labs/multichain-sync-btc/database"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/syncclient"
)

type FallBack struct {
	rpcClient      *syncclient.WalletBtcAccountClient
	db             *database.DB
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

func NewFallBack(cfg *config.Config, db *database.DB, rpcClient *syncclient.WalletBtcAccountClient, shutdown context.CancelCauseFunc) (*FallBack, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &FallBack{
		rpcClient:      rpcClient,
		db:             db,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in FallBacks: %w", err))
		}},
		ticker: time.NewTicker(cfg.ChainNode.WorkerInterval),
	}, nil
}

func (w *FallBack) Close() error {
	var result error
	w.resourceCancel()
	w.ticker.Stop()
	log.Info("stop fallback......")
	if err := w.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await FallBack %w", err))
		return result
	}
	log.Info("stop fallback success")
	return nil
}

func (w *FallBack) Start() error {
	log.Info("start fallback......")
	w.tasks.Go(func() error {
		for {
			select {
			case <-w.ticker.C:
				log.Info("fallback process start")
			case <-w.resourceCtx.Done():
				log.Info("stop fallback in worker")
				return nil
			}
		}
	})
	return nil
}
