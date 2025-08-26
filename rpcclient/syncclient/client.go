package syncclient

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/syncclient/common"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/syncclient/utxo"
)

type WalletBtcAccountClient struct {
	Ctx          context.Context
	ChainName    string
	BtcRpcClient utxo.WalletUtxoServiceClient
}

func NewWalletBtcAccountClient(ctx context.Context, rpc utxo.WalletUtxoServiceClient, chainName string) (*WalletBtcAccountClient, error) {
	log.Info("New account chain rpc client", "chainName", chainName)
	return &WalletBtcAccountClient{Ctx: ctx, BtcRpcClient: rpc, ChainName: chainName}, nil
}

func (wac *WalletBtcAccountClient) ExportAddressByPubKey(format, publicKey string) string {
	req := &utxo.ConvertAddressRequest{
		Format:    format,
		PublicKey: publicKey,
	}
	address, err := wac.BtcRpcClient.ConvertAddress(wac.Ctx, req)
	if address.Code == common.ReturnCode_ERROR {
		log.Error("covert address fail", "err", err)
		return ""
	}
	return address.Address
}

func (wac *WalletBtcAccountClient) GetBlockHeader(number *big.Int) (*BlockHeader, error) {
	request := &utxo.BlockHeaderNumberRequest{
		Network: "mainnet",
		Height:  number.Int64(),
	}
	blockHeader, err := wac.BtcRpcClient.GetBlockHeaderByNumber(context.Background(), request)
	if err != nil {
		return nil, err
	}
	blockNumber, _ := new(big.Int).SetString(blockHeader.Number, 10)

	return &BlockHeader{
		Hash:     blockHeader.BlockHash,
		PrevHash: blockHeader.ParentHash,
		Number:   blockNumber,
	}, nil
}

func (wac *WalletBtcAccountClient) GetBlockByNumber(blockNumber *big.Int) ([]*utxo.TransactionList, error) {
	blockReq := &utxo.BlockNumberRequest{
		Height: blockNumber.Int64(),
	}
	blockInfo, err := wac.BtcRpcClient.GetBlockByNumber(context.Background(), blockReq)
	if err != nil {
		log.Error("get block by number fail", "err", err)
		return nil, err
	}
	if blockInfo.Code == common.ReturnCode_ERROR {
		log.Error("Return code is error", "err", err)
	}
	return blockInfo.TxList, nil
}

func (wac *WalletBtcAccountClient) GetTransactionByHash(hash string) (*utxo.TxMessage, error) {
	return nil, nil
}

func (wac *WalletBtcAccountClient) GetAccount(address string) (int, error) {
	return 0, nil
}

func (wac *WalletBtcAccountClient) SendTx(rawTx string) (string, error) {
	return "", nil
}
