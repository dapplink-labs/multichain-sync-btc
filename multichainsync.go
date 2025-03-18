package multichain_transaction_syncs

import (
	"context"
	"github.com/dapplink-labs/multichain-sync-btc/bloomfilter"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/btc"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/config"
	"github.com/dapplink-labs/multichain-sync-btc/database"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient"
	"github.com/dapplink-labs/multichain-sync-btc/worker"
)

type MultiChainSync struct {
	Synchronizer *worker.BaseSynchronizer
	Deposit      *worker.Deposit
	Withdraw     *worker.Withdraw
	Internal     *worker.Internal
	BloomFilter  *bloomfilter.BloomFilter
	shutdown     context.CancelCauseFunc
	stopped      atomic.Bool
}

func NewMultiChainSync(ctx context.Context, cfg *config.Config, shutdown context.CancelCauseFunc) (*MultiChainSync, error) {
	db, err := database.NewDB(ctx, cfg.MasterDB)
	if err != nil {
		log.Error("init database fail", err)
		return nil, err
	}

	filter, err := bloomfilter.InitBloomFilter(ctx, &cfg.BloomFilter, db)
	if err != nil {
		log.Error("init bloom filter fail", err)
		return nil, err
	}

	log.Info("New deposit", "ChainAccountRpc", cfg.ChainBtcRpc)
	conn, err := grpc.NewClient(cfg.ChainBtcRpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("Connect to da retriever fail", "err", err)
		return nil, err
	}
	client := btc.NewWalletBtcServiceClient(conn)
	accountClient, err := rpcclient.NewWalletBtcAccountClient(context.Background(), client, "Ethereum")
	if err != nil {
		log.Error("new wallet account client fail", "err", err)
		return nil, err
	}

	deposit, _ := worker.NewDeposit(cfg, db, accountClient, shutdown)
	withdraw, _ := worker.NewWithdraw(cfg, db, accountClient, shutdown)
	internal, _ := worker.NewInternal(cfg, db, accountClient, shutdown)

	out := &MultiChainSync{
		Deposit:     deposit,
		Withdraw:    withdraw,
		Internal:    internal,
		BloomFilter: filter,
		shutdown:    shutdown,
	}
	return out, nil
}

func (mcs *MultiChainSync) Start(ctx context.Context) error {
	err := mcs.Deposit.Start()
	if err != nil {
		return err
	}
	err = mcs.Withdraw.Start()
	if err != nil {
		return err
	}
	err = mcs.Internal.Start()
	if err != nil {
		return err
	}
	return nil
}

func (mcs *MultiChainSync) Stop(ctx context.Context) error {
	err := mcs.Deposit.Close()
	if err != nil {
		return err
	}
	err = mcs.Withdraw.Close()
	if err != nil {
		return err
	}
	err = mcs.Internal.Close()
	if err != nil {
		return err
	}
	return nil
}

func (mcs *MultiChainSync) Stopped() bool {
	return mcs.stopped.Load()
}
