package rpcclient

import (
	"context"
	"github.com/ethereum/go-ethereum/log"
	"math/big"

	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/btc"
)

type WalletBtcAccountClient struct {
	Ctx          context.Context
	ChainName    string
	BtcRpcClient btc.WalletBtcServiceClient
}

func NewWalletBtcAccountClient(ctx context.Context, rpc btc.WalletBtcServiceClient, chainName string) (*WalletBtcAccountClient, error) {
	log.Info("New account chain rpc client", "chainName", chainName)
	return &WalletBtcAccountClient{Ctx: ctx, BtcRpcClient: rpc, ChainName: chainName}, nil
}

func (wac *WalletBtcAccountClient) ExportAddressByPubKey(format, publicKey string) string {
	req := &btc.ConvertAddressRequest{
		Format:    format,
		PublicKey: publicKey,
	}
	address, err := wac.BtcRpcClient.ConvertAddress(wac.Ctx, req)
	if address.Code == btc.ReturnCode_ERROR {
		log.Error("covert address fail", "err", err)
		return ""
	}
	return address.Address
}

func (wac *WalletBtcAccountClient) GetBlockHeader(number *big.Int) (*BlockHeader, error) {
	request := &btc.BlockHeaderNumberRequest{
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
		PrevHash: blockHeader.PrevHash,
		Number:   blockNumber,
	}, nil
}

func (wac *WalletBtcAccountClient) GetBlockByNumber(blockNumber *big.Int) ([]*btc.TransactionList, error) {
	blockReq := &btc.BlockNumberRequest{
		Height: blockNumber.Int64(),
	}
	blockInfo, err := wac.BtcRpcClient.GetBlockByNumber(context.Background(), blockReq)
	if err != nil {
		log.Error("get block by number fail", "err", err)
		return nil, err
	}
	if blockInfo.Code == btc.ReturnCode_ERROR {
		log.Error("Return code is error", "err", err)
	}
	return blockInfo.TxList, nil
}

func (wac *WalletBtcAccountClient) GetTransactionByHash(hash string) (*btc.TxMessage, error) {
	return nil, nil
}

func (wac *WalletBtcAccountClient) GetAccount(address string) (int, error) {
	return 0, nil
}

func (wac *WalletBtcAccountClient) SendTx(rawTx string) (string, error) {
	return "", nil
}
