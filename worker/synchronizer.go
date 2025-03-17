package worker

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/common/clock"
	"github.com/dapplink-labs/multichain-sync-btc/database"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/btc"
)

type Vin struct {
	Address string
	TxId    string
	Vout    uint8
	Amount  *big.Int
}

type Vout struct {
	Address string
	N       uint8
	Script  *btc.ScriptPubKey
	Amount  *big.Int
}

type Transaction struct {
	BusinessId  string
	BlockNumber *big.Int
	Hash        string
	TxFee       string
	TxType      string
	VinList     []Vin
	VoutList    []Vout
}

type Config struct {
	LoopIntervalMsec uint
	HeaderBufferSize uint
	StartHeight      *big.Int
	Confirmations    uint64
}

type BaseSynchronizer struct {
	loopInterval     time.Duration
	headerBufferSize uint64

	businessChannels chan map[string]*TransactionsChannel

	rpcClient  *rpcclient.WalletBtcAccountClient
	blockBatch *rpcclient.BatchBlock
	database   *database.DB

	headers []rpcclient.BlockHeader
	worker  *clock.LoopFn
}

type TransactionsChannel struct {
	BlockHeight  uint64
	ChannelId    string
	Transactions []*Transaction
}

func (syncer *BaseSynchronizer) Start() error {
	if syncer.worker != nil {
		return errors.New("already started")
	}
	syncer.worker = clock.NewLoopFn(clock.SystemClock, syncer.tick, func() error {
		log.Info("shutting down batch producer")
		close(syncer.businessChannels)
		return nil
	}, syncer.loopInterval)
	return nil
}

func (syncer *BaseSynchronizer) Close() error {
	if syncer.worker == nil {
		return nil
	}
	return syncer.worker.Close()
}

func (syncer *BaseSynchronizer) tick(_ context.Context) {
	if len(syncer.headers) > 0 {
		log.Info("retrying previous batch")
	} else {
		newHeaders, err := syncer.blockBatch.NextHeaders(syncer.headerBufferSize)
		if err != nil {
			log.Error("error querying for headers", "err", err)
		} else if len(newHeaders) == 0 {
			log.Warn("no new headers. syncer at head?")
		} else {
			syncer.headers = newHeaders
		}
	}
	err := syncer.processBatch(syncer.headers)
	if err == nil {
		syncer.headers = nil
	}
}

// 充值：  from 地址是外部地址；to 地址是系统数据的用户地址
// 提现：  from 地址热钱包地址；to 地址外部地址
// 归集：  from 地址是用户钱包地址，to 是热钱包地址
// 热转冷：from 地址是热钱包地址，to 是冷钱包地址
// 冷转热  from 地址是冷钱包地址，to 是热钱包地址
func (syncer *BaseSynchronizer) processBatch(headers []rpcclient.BlockHeader) error {
	if len(headers) == 0 {
		log.Info("headers is empty, no block waiting to handle")
		return nil
	}

	businessTxChannel := make(map[string]*TransactionsChannel)
	blockHeaders := make([]database.Blocks, len(headers))

	for i := range headers {
		log.Info("Sync block data", "height", headers[i].Number)
		blockHeaders[i] = database.Blocks{
			Hash:      headers[i].Hash,
			PrevHash:  headers[i].PrevHash,
			Number:    headers[i].Number,
			Timestamp: headers[i].Timestamp,
		}

		txList, err := syncer.rpcClient.GetBlockByNumber(blockHeaders[i].Number)
		if err != nil {
			return err
		}
		businessList, err := syncer.database.Business.QueryBusinessList()
		if err != nil {
			log.Error("query business list fail", "err", err)
			return err
		}
		for _, businessId := range businessList {
			var businessTransactions []*Transaction
			for _, tx := range txList {
				txItem := &Transaction{
					BusinessId:  businessId.BusinessUid,
					BlockNumber: headers[i].Number,
					Hash:        tx.Hash,
					TxFee:       tx.Fee,
					TxType:      "unknown",
				}
				var toAddressList []string
				var amountList []string
				var voutArray []Vout
				var vinArray []Vin
				for _, vout := range tx.Vout {
					toAddressList = append(toAddressList, vout.Address)
					amountList = append(amountList, big.NewInt(int64(vout.Amount)).String())
					voutItem := Vout{
						Address: vout.Address,
						N:       uint8(vout.Index),
						Script:  vout.ScriptPubKey,
						Amount:  big.NewInt(int64(vout.Amount)),
					}
					voutArray = append(voutArray, voutItem)
				}
				txItem.VoutList = voutArray
				var existToAddress bool
				var toAddressType uint8

				isDeposit, isWithdraw, isCollection, isToCold, isToHot := false, false, false, false, false
				for index := range toAddressList {
					existToAddress, toAddressType = syncer.database.Addresses.AddressExist(businessId.BusinessUid, toAddressList[index])
					hotWalletAddress, errHot := syncer.database.Addresses.QueryHotWalletInfo(businessId.BusinessUid)
					if errHot != nil {
						log.Error("Query hot wallet info", "err", err)
						return err
					}
					coldWalletAddress, errCold := syncer.database.Addresses.QueryColdWalletInfo(businessId.BusinessUid)
					if errCold != nil {
						log.Error("query cold wallet info fail", "err", err)
					}
					for _, txVin := range tx.Vin {
						vinItem := Vin{
							Address: txVin.Address,
							TxId:    tx.Hash,
							Vout:    uint8(txVin.Vout),
							Amount:  big.NewInt(int64(txVin.Amount)),
						}
						vinArray = append(vinArray, vinItem)
						addressList := strings.Split(txVin.Address, "|")
						for _, address := range addressList {
							vinAddress, errQuery := syncer.database.Addresses.QueryAddressesByToAddress(businessId.BusinessUid, address)
							if errQuery != nil {
								log.Error("Query address fail", "err", err)
								return err
							}
							if vinAddress == nil && existToAddress && toAddressType == 0 {
								isDeposit = true
							}
							if existToAddress && toAddressType == 1 && vinAddress != nil {
								isCollection = true
							}
							if address == hotWalletAddress.Address && !existToAddress {
								isWithdraw = true
							}
							if existToAddress && toAddressType == 2 && address == hotWalletAddress.Address {
								isToCold = true
							}
							if address == coldWalletAddress.Address && existToAddress && toAddressType == 1 {
								isToHot = true
							}
						}
					}
				}
				// 对于一笔交易来说，出金的地址相对于入金的地址来说是 vout; 入金的地址相对于出金地址来说他是 vin
				if isDeposit { // 充值，to 地址是用户地址代表充值, 通过 txid 和地址来匹配一个 vin
					txItem.TxType = "deposit"
				}

				if isWithdraw { // 提现
					txItem.TxType = "withdraw"
				}

				if isCollection { // 归集； 1: 代表热钱包地址
					txItem.TxType = "collection"
				}

				if isToCold { // 热转冷；2 是冷钱包地址
					txItem.TxType = "hot2cold"
				}

				if isToHot { // 冷转热；
					txItem.TxType = "cold2hot"
				}
				businessTransactions = append(businessTransactions, txItem)
			}
			if len(businessTransactions) > 0 {
				if businessTxChannel[businessId.BusinessUid] == nil {
					businessTxChannel[businessId.BusinessUid] = &TransactionsChannel{
						BlockHeight:  headers[i].Number.Uint64(),
						Transactions: businessTransactions,
					}
				} else {
					businessTxChannel[businessId.BusinessUid].BlockHeight = headers[i].Number.Uint64()
					businessTxChannel[businessId.BusinessUid].Transactions = append(businessTxChannel[businessId.BusinessUid].Transactions, businessTransactions...)
				}
			}
		}
	}
	return nil
}
