package services

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/database"
	"github.com/dapplink-labs/multichain-sync-btc/database/dynamic"
	dal_wallet_go "github.com/dapplink-labs/multichain-sync-btc/protobuf/dal-wallet-go"
	"github.com/dapplink-labs/multichain-sync-btc/rpcclient/syncclient/utxo"
)

const (
	ConsumerToken = "DappLink123456"
)

func (bws *BusinessMiddleWireServices) BusinessRegister(ctx context.Context, request *dal_wallet_go.BusinessRegisterRequest) (*dal_wallet_go.BusinessRegisterResponse, error) {
	if request.RequestId == "" || request.NotifyUrl == "" {
		return &dal_wallet_go.BusinessRegisterResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "invalid params",
		}, nil
	}
	business := &database.Business{
		GUID:        uuid.New(),
		BusinessUid: request.RequestId,
		NotifyUrl:   request.NotifyUrl,
		Timestamp:   uint64(time.Now().Unix()),
	}
	err := bws.db.Business.StoreBusiness(business)
	if err != nil {
		log.Error("store business fail", "err", err)
		return &dal_wallet_go.BusinessRegisterResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "store db fail",
		}, nil
	}
	dynamic.CreateTableFromTemplate(request.RequestId, bws.db)
	return &dal_wallet_go.BusinessRegisterResponse{
		Code: dal_wallet_go.ReturnCode_SUCCESS,
		Msg:  "config business success",
	}, nil
}

func (bws *BusinessMiddleWireServices) ExportAddressesByPublicKeys(ctx context.Context, request *dal_wallet_go.ExportAddressesRequest) (*dal_wallet_go.ExportAddressesResponse, error) {
	var (
		retAddressess []*dal_wallet_go.Address
		dbAddresses   []database.Addresses
		balances      []database.Balances
	)
	for _, value := range request.PublicKeys {
		address := bws.syncClient.ExportAddressByPubKey(value.Format, value.PublicKey)
		item := &dal_wallet_go.Address{
			Type:    value.Type,
			Address: address,
		}
		dbAddress := database.Addresses{
			GUID:        uuid.New(),
			Address:     address,
			AddressType: uint8(value.Type),
			PublicKey:   value.PublicKey,
			Timestamp:   uint64(time.Now().Unix()),
		}
		dbAddresses = append(dbAddresses, dbAddress)

		balanceItem := database.Balances{
			GUID:        uuid.New(),
			Address:     address,
			AddressType: uint8(value.Type),
			Balance:     big.NewInt(0),
			LockBalance: big.NewInt(0),
			Timestamp:   uint64(time.Now().Unix()),
		}
		balances = append(balances, balanceItem)

		retAddressess = append(retAddressess, item)
	}
	err := bws.db.Addresses.StoreAddresses(request.RequestId, dbAddresses)
	if err != nil {
		return &dal_wallet_go.ExportAddressesResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "store address to db fail",
		}, nil
	}
	err = bws.db.Balances.StoreBalances(request.RequestId, balances)
	if err != nil {
		return &dal_wallet_go.ExportAddressesResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "store balance to db fail",
		}, nil
	}
	return &dal_wallet_go.ExportAddressesResponse{
		Code:      dal_wallet_go.ReturnCode_SUCCESS,
		Msg:       "generate address success",
		Addresses: retAddressess,
	}, nil
}

func (bws *BusinessMiddleWireServices) BuildUnSignTransaction(ctx context.Context, request *dal_wallet_go.UnSignWithdrawTransactionRequest) (*dal_wallet_go.UnSignWithdrawTransactionResponse, error) {
	if err := validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	amountBig, ok := new(big.Int).SetString(request.Txn[0].Value, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount value: %s", request.Txn[0].Value)
	}

	feeReq := &utxo.FeeRequest{
		ConsumerToken: ConsumerToken,
		Chain:         bws.BusinessMiddleConfig.ChainName,
		Network:       bws.BusinessMiddleConfig.NetWork,
		Coin:          bws.BusinessMiddleConfig.CoinName,
		RawTx:         "",
	}

	utxoFee, err := bws.syncClient.BtcRpcClient.GetFee(context.Background(), feeReq)
	if err != nil {
		log.Error("get btc fee fail", "err", err)
		return nil, err
	}

	btcSt := utxoFee.FeeRate * 10e8

	btcStStr := fmt.Sprintf("%f", btcSt) // 每个字节消耗手续费聪

	/*
	 根据是 taproot，隔离见证或者legacy 的预估单个 input 和 output 字节数，再根据 input 和 output 的数量做出总字节数
	 再去乘以单个字节需要消耗聪的手续，得到的就是这笔交易的手续费

	 如果铭文和符石，直接先进行一次预签名进行，铭文和符石，一个 witness, 一个 op-return, 不管是在那个结构里面都是要消耗的手续
	*/

	btcStBigIntFee, _ := new(big.Int).SetString(btcStStr, 10)

	utr := &utxo.UnSignTransactionRequest{
		ConsumerToken: ConsumerToken,
		Chain:         bws.BusinessMiddleConfig.ChainName,
		Network:       bws.BusinessMiddleConfig.NetWork,
		Fee:           btcStStr, // 每个字节消耗手续费聪
	}

	txMessageHash, err := bws.syncClient.BtcRpcClient.CreateUnSignTransaction(context.Background(), utr)
	if err != nil {
		log.Error("create un sign transaction fail", "err", err)
		return nil, err
	}
	log.Info("txMessageHash", "txMessageHash", txMessageHash)
	if err := bws.storeWithdraw(request, amountBig, btcStBigIntFee); err != nil {
		return nil, fmt.Errorf("store withdraw failed: %w", err)
	}

	return nil, nil
}

func (bws *BusinessMiddleWireServices) BuildSignedTransaction(ctx context.Context, request *dal_wallet_go.SignedWithdrawTransactionRequest) (*dal_wallet_go.SignedWithdrawTransactionResponse, error) {
	return nil, nil
}

func validateRequest(request *dal_wallet_go.UnSignWithdrawTransactionRequest) error {
	return nil
}

func (bws *BusinessMiddleWireServices) storeWithdraw(request *dal_wallet_go.UnSignWithdrawTransactionRequest, amountBig *big.Int, fee *big.Int) error {
	withdraw := &database.Withdraws{
		Guid:        uuid.New(),
		Timestamp:   uint64(time.Now().Unix()),
		Status:      uint8(database.TxStatusUnsigned),
		BlockHash:   "",
		BlockNumber: big.NewInt(1),
		Hash:        "",
		Fee:         amountBig,
		TxSignHex:   "",
	}
	return bws.db.Withdraws.StoreWithdraws(request.RequestId, withdraw)
}

func (bws *BusinessMiddleWireServices) storeInternal(request *dal_wallet_go.UnSignWithdrawTransactionRequest, amountBig *big.Int, fee *big.Int) error {
	internal := &database.Internals{
		Guid:        uuid.New(),
		Timestamp:   uint64(time.Now().Unix()),
		Status:      uint8(database.TxStatusUnsigned),
		BlockHash:   "",
		BlockNumber: big.NewInt(1),
		Hash:        "",
		Fee:         fee,
		TxSignHex:   "",
	}
	return bws.db.Internals.StoreInternal(request.RequestId, internal)
}
