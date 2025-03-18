package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/common/retry"
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
	businessList, err := deposit.database.Business.QueryBusinessList()
	if err != nil {
		log.Error("query business list fail", "err", err)
		return err
	}
	for _, business := range businessList {
		_, exists := batch[business.BusinessUid]
		if !exists {
			continue
		}

		var (
			transactionFlowList []database.Transactions
			depositList         []database.Deposits
			withdrawList        []database.Withdraws
			internals           []database.Internals
			vins                []database.Vins
			vouts               []database.Vouts
			balances            []database.TokenBalance
		)

		log.Info("handle business flow", "businessId", business.BusinessUid, "chainLatestBlock", batch[business.BusinessUid].BlockHeight, "txn", len(batch[business.BusinessUid].Transactions))
		var pvList []*PrepareVoutList
		for _, tx := range batch[business.BusinessUid].Transactions {

			txItem, err := deposit.rpcClient.GetTransactionByHash(tx.Hash)
			if err != nil {
				log.Info("get transaction by hash fail", "err", err)
				return err
			}

			log.Info("get transaction success", "txHash", txItem.Hash)
			transactionFlow, err := deposit.HandleTransaction(tx)
			if err != nil {
				log.Info("handle  transaction fail", "err", err)
				return err
			}
			transactionFlowList = append(transactionFlowList, transactionFlow)

			vintListPre, vinBalances, err := deposit.HandleVin(tx)
			if err != nil {
				log.Error("handle vout fail", "err", err)
			}
			vins = append(vins, vintListPre...)
			balances = append(balances, vinBalances...)

			voutListPre, voutBalances, err := deposit.HandleVout(tx, business.BusinessUid)
			if err != nil {
				log.Error("handle vout fail", "err", err)
			}
			balances = append(balances, voutBalances...)

			pvList = append(pvList, voutListPre)
			vlist := voutListPre.VoutList
			vouts = append(vouts, vlist...)

			switch tx.TxType {
			case "deposit":
				depositItem, _ := deposit.HandleDeposit(tx)
				depositList = append(depositList, depositItem)
				break
			case "withdraw":
				withdrawItem, _ := deposit.HandleWithdraw(tx)
				withdrawList = append(withdrawList, withdrawItem)
				break
			case "collection", "hot2cold", "cold2hot":
				internelItem, _ := deposit.HandleInternalTx(tx)
				internals = append(internals, internelItem)
				break
			default:
				break
			}
		}
		retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
		if _, err := retry.Do[interface{}](deposit.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
			if err := deposit.database.Transaction(func(tx *database.DB) error {
				if len(depositList) > 0 {
					log.Info("Store deposit transaction success", "totalTx", len(depositList))
					if err := tx.Deposits.StoreDeposits(business.BusinessUid, depositList); err != nil {
						return err
					}
				}

				if err := tx.Deposits.UpdateDepositsComfirms(business.BusinessUid, batch[business.BusinessUid].BlockHeight, uint64(deposit.confirms)); err != nil {
					log.Info("Handle confims fail", "totalTx", "err", err)
					return err
				}

				if len(balances) > 0 {
					log.Info("Handle balances success", "totalTx", len(balances))
					if err := tx.Balances.UpdateOrCreate(business.BusinessUid, balances); err != nil {
						return err
					}
				}

				if len(withdrawList) > 0 {
					if err := tx.Withdraws.UpdateWithdrawStatus(business.BusinessUid, database.TxStatusWalletDone, withdrawList); err != nil {
						return err
					}
				}

				if len(internals) > 0 {
					if err := tx.Internals.UpdateInternalStatus(business.BusinessUid, database.TxStatusWalletDone, internals); err != nil {
						return err
					}
				}
				if len(transactionFlowList) > 0 {
					if err := tx.Transactions.StoreTransactions(business.BusinessUid, transactionFlowList); err != nil {
						return err
					}
				}
				if len(vins) > 0 {
					if err := tx.Vins.StoreVins(business.BusinessUid, vins); err != nil {
						return err
					}
				}
				if len(vouts) > 0 {
					if err := tx.Vouts.StoreVouts(business.BusinessUid, vouts); err != nil {
						return err
					}
				}
				if len(pvList) > 0 {
					for _, pvItem := range pvList {
						for _, voutItmepv := range pvItem.VoutList {
							if err := tx.Vins.UpdateVinsTx(business.BusinessUid, pvItem.TxId, voutItmepv.Address, true, pvItem.TxId, pvItem.BlockNumber); err != nil {
								return err
							}
						}
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
	return nil
}

func (deposit *Deposit) HandleDeposit(tx *Transaction) (database.Deposits, error) {
	var addressString string
	var amountStrig string
	for _, voutItem := range tx.VoutList {
		addressString += "|" + voutItem.Address
		amountStrig += "|" + voutItem.Amount.String()
	}
	txFee, _ := new(big.Int).SetString(tx.TxFee, 10)
	depositTx := database.Deposits{
		GUID:        uuid.New(),
		BlockHash:   "",
		BlockNumber: tx.BlockNumber,
		Hash:        tx.Hash,
		FromAddress: "",
		ToAddress:   addressString,
		Fee:         txFee,
		Amount:      amountStrig,
		Status:      0,
		Timestamp:   uint64(time.Now().Unix()),
	}
	return depositTx, nil
}

func (deposit *Deposit) HandleWithdraw(tx *Transaction) (database.Withdraws, error) {
	txFee, _ := new(big.Int).SetString(tx.TxFee, 10)
	var addressString string
	var amountString string
	for _, vinItem := range tx.VinList {
		addressString += "|" + vinItem.Address
		// 根据 txid 和 address 查询 vin 表确定金额
		addressList := strings.Split(vinItem.Address, "|")
		for _, addressItem := range addressList {
			vinTx, _ := deposit.database.Vins.QueryVinByTxId(tx.BusinessId, addressItem, tx.Hash)
			amountString += "|" + vinTx.Amount.String()
		}
	}
	var addressToString string
	for _, voutItem := range tx.VoutList {
		addressToString += "|" + voutItem.Address
	}
	withdrawTx := database.Withdraws{
		Guid:        uuid.New(),
		BlockHash:   "",
		BlockNumber: tx.BlockNumber,
		Hash:        tx.Hash,
		FromAddress: addressString,
		ToAddress:   addressToString,
		Amount:      amountString,
		Fee:         txFee,
		Status:      uint8(database.TxStatusWalletDone),
		Timestamp:   uint64(time.Now().Unix()),
	}
	return withdrawTx, nil
}

func (deposit *Deposit) HandleTransaction(tx *Transaction) (database.Transactions, error) {
	txFee, _ := new(big.Int).SetString(tx.TxFee, 10)
	var fromAddressString, toAddressString, amountString string
	if tx.TxType == "deposit" {
		for _, voutItem := range tx.VoutList {
			fromAddressString += "|" + voutItem.Address
			amountString += "|" + voutItem.Amount.String()
		}
	}

	if tx.TxType == "withdraw" {
		for _, vinItem := range tx.VinList {
			toAddressString += "|" + vinItem.Address
			addressList := strings.Split(vinItem.Address, "|")
			for _, addressItem := range addressList {
				vinTx, _ := deposit.database.Vins.QueryVinByTxId(tx.BusinessId, addressItem, tx.Hash)
				amountString += "|" + vinTx.Amount.String()
			}
		}
	}

	transactionTx := database.Transactions{
		GUID:        uuid.New(),
		BlockHash:   "",
		BlockNumber: tx.BlockNumber,
		Hash:        tx.Hash,
		FromAddress: fromAddressString,
		ToAddress:   toAddressString,
		Fee:         txFee,
		Status:      0,
		Amount:      amountString,
		TxType:      tx.TxType,
		Timestamp:   uint64(time.Now().Unix()),
	}
	return transactionTx, nil
}

func (deposit *Deposit) HandleInternalTx(tx *Transaction) (database.Internals, error) {
	txFee, _ := new(big.Int).SetString(tx.TxFee, 10)
	var amountString string
	var fromAddressString, toAddressString, userAmountString, hotAmountString, coldAmountSring string
	if tx.TxType == "collection" { // 用户地址到热钱包地址
		for _, voutItem := range tx.VoutList {
			fromAddressString += "|" + voutItem.Address
			userAmountString += "|" + voutItem.Amount.String()
		}

		for _, vinItem := range tx.VinList {
			toAddressString += "|" + vinItem.Address
			addressList := strings.Split(vinItem.Address, "|")
			for _, addressItem := range addressList {
				vinTx, _ := deposit.database.Vins.QueryVinByTxId(tx.BusinessId, addressItem, tx.Hash)
				hotAmountString += "|" + vinTx.Amount.String()
			}
		}
		amountString = "user" + userAmountString + "hotwallet" + hotAmountString
	}

	if tx.TxType == "hot2cold" { // 热转冷
		for _, voutItem := range tx.VoutList {
			fromAddressString += "|" + voutItem.Address
			hotAmountString += "|" + voutItem.Amount.String()
		}
		for _, vinItem := range tx.VinList {
			toAddressString += "|" + vinItem.Address
			addressList := strings.Split(vinItem.Address, "|")
			for _, addressItem := range addressList {
				vinTx, _ := deposit.database.Vins.QueryVinByTxId(tx.BusinessId, addressItem, tx.Hash)
				coldAmountSring += "|" + vinTx.Amount.String()
			}
		}
		amountString = "hotwallet" + hotAmountString + "coldtwallet" + coldAmountSring
	}
	if tx.TxType == "cold2hot" { // 冷转热  to
		for _, voutItem := range tx.VoutList {
			fromAddressString += "|" + voutItem.Address
			coldAmountSring += "|" + voutItem.Amount.String()
		}

		for _, vinItem := range tx.VinList {
			toAddressString += "|" + vinItem.Address
			addressList := strings.Split(vinItem.Address, "|")
			for _, addressItem := range addressList {
				vinTx, _ := deposit.database.Vins.QueryVinByTxId(tx.BusinessId, addressItem, tx.Hash)
				hotAmountString += "|" + vinTx.Amount.String()
			}
		}
		amountString = "coldtwallet" + coldAmountSring + "hotwallet" + hotAmountString
	}
	internalTx := database.Internals{
		Guid:        uuid.New(),
		BlockHash:   "",
		BlockNumber: tx.BlockNumber,
		Hash:        tx.Hash,
		FromAddress: fromAddressString,
		ToAddress:   toAddressString,
		Status:      1,
		Amount:      amountString,
		Fee:         txFee,
		Timestamp:   uint64(time.Now().Unix()),
	}
	return internalTx, nil
}

func (deposit *Deposit) HandleVin(tx *Transaction) ([]database.Vins, []database.TokenBalance, error) {
	var vinList []database.Vins
	var balanceList []database.TokenBalance
	for _, vout := range tx.VoutList {
		vinTx := database.Vins{
			GUID:             uuid.New(),
			Address:          vout.Address,
			TxId:             tx.Hash,
			Vout:             vout.N,
			Script:           "",
			Witness:          "",
			Amount:           vout.Amount,
			SpendTxHash:      "",
			SpendBlockHeight: big.NewInt(0),
			IsSpend:          false,
			Timestamp:        uint64(time.Now().Unix()),
		}
		if tx.TxType == "deposit" || tx.TxType == "collection" || tx.TxType == "hot2cold" || tx.TxType == "cold2hot" {
			balanceItem := database.TokenBalance{
				FromAddress:  "",
				ToAddress:    vout.Address,
				TokenAddress: "",
				Balance:      vout.Amount,
				TxType:       tx.TxType,
			}
			balanceList = append(balanceList, balanceItem)
		}
		vinList = append(vinList, vinTx)
	}
	return vinList, balanceList, nil
}

func (deposit *Deposit) HandleVout(tx *Transaction, businessID string) (*PrepareVoutList, []database.TokenBalance, error) {
	var voutList []database.Vouts
	var balanceList []database.TokenBalance
	for _, vin := range tx.VinList {
		vout := database.Vouts{
			GUID:      uuid.New(),
			Address:   vin.Address,
			N:         vin.Vout,
			Amount:    vin.Amount,
			Timestamp: uint64(time.Now().Unix()),
		}
		voutList = append(voutList, vout)
		if tx.TxType == "withdraw" || tx.TxType == "collection" || tx.TxType == "hot2cold" || tx.TxType == "cold2hot" {
			vinAddressess := strings.Split(vin.Address, "|")
			for _, addr := range vinAddressess {
				vinDetail, err := deposit.database.Vins.QueryVinByTxId(businessID, addr, tx.Hash)
				if err != nil {
					log.Error("query vins fail", "err", err)
				}
				balanceItem := database.TokenBalance{
					FromAddress:  addr,
					ToAddress:    "",
					TokenAddress: "",
					Balance:      vinDetail.Amount,
					TxType:       tx.TxType,
				}
				balanceList = append(balanceList, balanceItem)
			}
		}

	}
	return &PrepareVoutList{
		TxId:        tx.Hash,
		BlockNumber: tx.BlockNumber,
		VoutList:    voutList,
	}, balanceList, nil
}

type PrepareVoutList struct {
	TxId        string
	BlockNumber *big.Int
	VoutList    []database.Vouts
}
