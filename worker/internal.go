package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/dapplink-labs/multichain-sync-btc/common/retry"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/common/tasks"
	"github.com/dapplink-labs/multichain-sync-btc/config"
	"github.com/dapplink-labs/multichain-sync-btc/database"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/syncclient"
)

type Internal struct {
	rpcClient      *syncclient.WalletBtcAccountClient
	db             *database.DB
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

func NewInternal(cfg *config.Config, db *database.DB, rpcClient *syncclient.WalletBtcAccountClient, shutdown context.CancelCauseFunc) (*Internal, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Internal{
		rpcClient:      rpcClient,
		db:             db,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in internals: %w", err))
		}},
		ticker: time.NewTicker(cfg.ChainNode.WorkerInterval),
	}, nil
}

func (w *Internal) Close() error {
	var result error
	w.resourceCancel()
	w.ticker.Stop()
	log.Info("stop internal......")
	if err := w.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await internal %w", err))
		return result
	}
	log.Info("stop internal success")
	return nil
}

func (w *Internal) Start() error {
	log.Info("start internals......")
	w.tasks.Go(func() error {
		for {
			select {
			case <-w.ticker.C:
				log.Info("collection and hot to cold")
				businessList, err := w.db.Business.QueryBusinessList()
				if err != nil {
					log.Error("query business list fail", "err", err)
					continue
				}

				for _, businessId := range businessList {
					unSendInternalTxList, err := w.db.Internals.UnSendInternalsList(businessId.BusinessUid)
					if err != nil {
						log.Error("query un send internal tx list fail", "err", err)
						continue
					}
					var balanceList []database.Balances
					for _, unSendInternalTx := range unSendInternalTxList {
						//bAddressList := strings.Split(unSendInternalTx.FromAddress, "|")
						//bAmountList := strings.Split(unSendInternalTx.Amount, "|")
						//for index, _ := range bAddressList {
						//	lockBalance, _ := new(big.Int).SetString(bAmountList[index], 10)
						//	balanceItem := database.Balances{
						//		Address:     bAddressList[index],
						//		LockBalance: lockBalance,
						//	}
						//	balanceList = append(balanceList, balanceItem)
						//}
						txHash, err := w.rpcClient.SendTx(unSendInternalTx.TxSignHex)
						if err != nil {
							log.Error("send transaction fail", "err", err)
							continue
						} else {
							unSendInternalTx.Hash = txHash
							unSendInternalTx.Status = uint8(database.TxStatusBroadcasted)
						}
					}

					retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
					if _, err := retry.Do[interface{}](w.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
						if err := w.db.Transaction(func(tx *database.DB) error {
							if len(balanceList) > 0 {
								log.Info("Update address balance", "totalTx", len(balanceList))
								if err := tx.Balances.UpdateBalances(businessId.BusinessUid, balanceList); err != nil {
									log.Error("Update address balance fail", "err", err)
									return err
								}
							}
							if len(unSendInternalTxList) > 0 {
								err = w.db.Internals.UpdateInternalStatus(businessId.BusinessUid, database.TxStatusWalletDone, unSendInternalTxList)
								if err != nil {
									log.Error("update internals status fail", "err", err)
									return err
								}
							}
							return nil
						}); err != nil {
							log.Error("unable to persist batch", "err", err)
							return nil, err
						}
						return nil, nil
					}); err != nil {
						return err
					}
				}
			case <-w.resourceCtx.Done():
				log.Info("stop internals in worker")
				return nil
			}
		}
	})
	return nil
}
