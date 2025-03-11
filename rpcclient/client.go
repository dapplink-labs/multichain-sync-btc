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
	return nil, nil
}

func (wac *WalletBtcAccountClient) GetBlockInfo(blockNumber *big.Int) ([]*btc.TxMessage, error) {
	return nil, nil
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
