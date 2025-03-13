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
		address := bws.btcClient.ExportAddressByPubKey(value.Format, value.PublicKey)
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
	// todo: fee docking to fee 预估器
	fee := big.NewInt(0)
	switch request.Txn[0].TxType {
	case "withdraw":
		if err := bws.storeWithdraw(request, amountBig, fee); err != nil {
			return nil, fmt.Errorf("store withdraw failed: %w", err)
		}
	case "collection", "hot2cold":
		if err := bws.storeInternal(request, amountBig, fee); err != nil {
			return nil, fmt.Errorf("store internal transaction failed: %w", err)
		}
	default:
		fmt.Println("aaa")
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
	var withdraws []database.Withdraws
	withdraw := database.Withdraws{
		Guid:        request.Txn[0].TransactionUuid,
		Timestamp:   uint64(time.Now().Unix()),
		Status:      uint8(database.TxStatusUnsigned),
		BlockHash:   "",
		BlockNumber: big.NewInt(1),
		Hash:        "",
		FromAddress: request.Txn[0].From,
		ToAddress:   request.Txn[0].To,
		Amount:      amountBig.String(),
		Fee:         amountBig,
		TxSignHex:   "",
	}
	withdraws = append(withdraws, withdraw)
	return bws.db.Withdraws.StoreWithdraws(request.RequestId, withdraws)
}

func (bws *BusinessMiddleWireServices) storeInternal(request *dal_wallet_go.UnSignWithdrawTransactionRequest, amountBig *big.Int, fee *big.Int) error {
	var internals []database.Internals
	internal := database.Internals{
		Guid:        request.Txn[0].TransactionUuid,
		Timestamp:   uint64(time.Now().Unix()),
		Status:      uint8(database.TxStatusUnsigned),
		BlockHash:   "",
		BlockNumber: big.NewInt(1),
		Hash:        "",
		FromAddress: request.Txn[0].From,
		ToAddress:   request.Txn[0].To,
		Amount:      amountBig.String(),
		Fee:         fee,
		TxSignHex:   "",
	}
	internals = append(internals, internal)
	return bws.db.Internals.StoreInternals(request.RequestId, internals)
}
