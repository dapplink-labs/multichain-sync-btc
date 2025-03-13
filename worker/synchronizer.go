package worker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/common/clock"
	"github.com/dapplink-labs/multichain-sync-btc/database"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient"
)

type Transaction struct {
	BusinessId  string
	BlockNumber *big.Int
	TxType      string
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

		for _, tx := range txList {
			fmt.Println(tx.GetVin())
		}
	}
	if len(businessTxChannel) > 0 {
		syncer.businessChannels <- businessTxChannel
	}
	return nil
}
