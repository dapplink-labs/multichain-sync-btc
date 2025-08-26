package signclient

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/signclient/wallet"
)

type SignMachineRpcClient struct {
	Ctx          context.Context
	ChainName    string
	NetWork      string
	SignRpClient wallet.WalletServiceClient
}

func NewSignMachineRpcClient(ctx context.Context, signRpClient wallet.WalletServiceClient, chainName string) (*SignMachineRpcClient, error) {
	log.Info("New account chain rpc http", "chainName", chainName)
	return &SignMachineRpcClient{Ctx: ctx, SignRpClient: signRpClient, ChainName: chainName}, nil
}

func (smr *SignMachineRpcClient) BuildAndSignTransaction(publicKey string, walletKeyHash string, riskKeyHash string, txBase64Body string) (*SignedTransaction, error) {
	signRequest := &wallet.BuildAndSignTransactionRequest{
		ConsumerToken: "DappLink123456",
		ChainName:     smr.ChainName,
		Network:       smr.NetWork,
		PublicKey:     publicKey,
		WalletKeyHash: walletKeyHash,
		RiskKeyHash:   riskKeyHash,
		TxBase64Body:  txBase64Body,
	}
	signedTxn, err := smr.SignRpClient.BuildAndSignTransaction(smr.Ctx, signRequest)
	if err != nil {
		log.Error("build and sign transaction fail", "err", err)
		return &SignedTransaction{}, err
	}
	if signedTxn.Code == wallet.ReturnCode_ERROR {
		return &SignedTransaction{}, err
	}
	return &SignedTransaction{
		TxMessageHash: signedTxn.TxMessageHash,
		TxHash:        signedTxn.TxHash,
		SignedTx:      signedTxn.SignedTx,
	}, nil
}
