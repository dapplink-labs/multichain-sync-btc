package services

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
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
	resp := &dal_wallet_go.UnSignWithdrawTransactionResponse{
		Code: dal_wallet_go.ReturnCode_ERROR,
		Msg:  "submit withdraw fail",
	}
	if request.ConsumerToken != ConsumerToken {
		resp.Msg = "consumer token is error"
		return resp, nil
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
	btcSt := utxoFee.FeeRate * 10e8 * 380
	btcStStr := fmt.Sprintf("%f", btcSt) // 每个字节消耗手续费聪

	/*
	 根据是 taproot，隔离见证或者legacy 的预估单个 input 和 output 字节数，再根据 input 和 output 的数量做出总字节数
	 再去乘以单个字节需要消耗聪的手续，得到的就是这笔交易的手续费

	 如果铭文和符石，直接先进行一次预签名进行，铭文和符石，一个 witness, 一个 op-return, 不管是在那个结构里面都是要消耗的手续
	*/
	// todo: 上面预测模型实现

	btcStBigIntFee, _ := new(big.Int).SetString(btcStStr, 10)

	howWalletInfo, err := bws.db.Addresses.QueryHotWalletInfo(request.RequestId)
	if err != nil {
		log.Error("query hot wallet info fail", "err", err)
		return nil, err
	}

	// todo: 需要找到和提现交易匹配的热钱包地址的 vin,
	// - 暴力的形式，将所有 utxo 输入进去，然后找零
	// - 将 uxto 进行排序，选择和提现相近的交易放到 vin, 此种方案会造成 utxo 臃肿，需要通过合并 utxo 解决臃肿问题
	vinsList, err := bws.db.Vins.QueryVinsByAddress(request.RequestId, howWalletInfo.Address)
	if err != nil {
		return nil, err
	}

	var utxoVins []*utxo.Vin
	for _, dbVin := range vinsList {
		vinItem := &utxo.Vin{
			Hash:    dbVin.TxId,
			Index:   uint32(dbVin.Vout),
			Amount:  dbVin.Amount.Int64(),
			Address: howWalletInfo.Address,
		}
		utxoVins = append(utxoVins, vinItem)
	}

	var utxoVouts []*utxo.Vout

	for _, reqVout := range request.Txn {
		aomumt, _ := strconv.Atoi(reqVout.Value)
		voutItem := &utxo.Vout{
			Address: reqVout.To,
			Amount:  int64(aomumt),
			Index:   0,
		}
		utxoVouts = append(utxoVouts, voutItem)
	}

	utr := &utxo.UnSignTransactionRequest{
		ConsumerToken: ConsumerToken,
		Chain:         bws.BusinessMiddleConfig.ChainName,
		Network:       bws.BusinessMiddleConfig.NetWork,
		Fee:           btcStStr, // 每个字节消耗手续费聪
		Vin:           utxoVins,
		Vout:          utxoVouts,
	}

	txMessageHash, err := bws.syncClient.BtcRpcClient.CreateUnSignTransaction(context.Background(), utr)
	if err != nil {
		log.Error("create un sign transaction fail", "err", err)
		return nil, err
	}
	log.Info("txMessageHash", "txMessageHash", txMessageHash)

	transactionUuid := uuid.New()
	withdraw := &database.Withdraws{
		Guid:        transactionUuid,
		BlockHash:   "0x0",
		BlockNumber: big.NewInt(0),
		Hash:        "0x0",
		Fee:         btcStBigIntFee,
		LockTime:    big.NewInt(0),
		Version:     "0x0",
		TxSignHex:   "0x0",
		Status:      database.TxStatusWaitSign,
		Timestamp:   uint64(time.Now().Unix()),
	}
	err = bws.db.Withdraws.StoreWithdraws(request.RequestId, withdraw)
	if err != nil {
		log.Error("store withdraws fail", "err", err)
		return nil, err
	}
	resp.Code = dal_wallet_go.ReturnCode_SUCCESS
	resp.Msg = "create tx message hash success"
	var retTxHashList []*dal_wallet_go.ReturnTransactionHashes
	var SignHashesStr []string
	for _, b := range txMessageHash.SignHashes {
		if b != nil {
			SignHashesStr = append(SignHashesStr, string(b))
		} else {
			SignHashesStr = append(SignHashesStr, "")
		}
	}
	var SignHashStr string
	for _, msg := range SignHashesStr {
		SignHashStr += msg + "|"
	}
	retHash := &dal_wallet_go.ReturnTransactionHashes{
		TransactionUuid: transactionUuid.String(),
		UnSignTx:        SignHashStr,
		TxData:          string(txMessageHash.TxData),
	}
	retTxHashList = append(retTxHashList, retHash)
	resp.ReturnTxHashes = retTxHashList
	return resp, nil
}

func (bws *BusinessMiddleWireServices) BuildSignedTransaction(ctx context.Context, request *dal_wallet_go.SignedWithdrawTransactionRequest) (*dal_wallet_go.SignedWithdrawTransactionResponse, error) {
	resp := &dal_wallet_go.SignedWithdrawTransactionResponse{
		Code: dal_wallet_go.ReturnCode_ERROR,
		Msg:  "submit withdraw fail",
	}
	if request.ConsumerToken != ConsumerToken {
		resp.Msg = "consumer token is error"
		return resp, nil
	}

	var resultSignature [][]byte
	var txData []byte
	var transactionId string
	for _, SignTx := range request.SignTxn {
		if SignTx != nil {
			signatureItem := SignTx.Signature
			resultSignature = append(resultSignature, []byte(signatureItem))
		} else {
			resultSignature = append(resultSignature, nil)
		}
		txData = []byte(SignTx.TxData)
		transactionId = SignTx.TransactionUuid
	}
	hotWalletInfo, err := bws.db.Addresses.QueryHotWalletInfo(request.RequestId)
	if err != nil {
		return nil, err
	}
	var publicKeys [][]byte
	publicKeys = append(publicKeys, []byte(hotWalletInfo.PublicKey))
	signedReq := &utxo.SignedTransactionRequest{
		ConsumerToken: "ConsumerToken",
		Chain:         bws.ChainName,
		Network:       bws.NetWork,
		TxData:        txData,
		Signatures:    resultSignature,
		PublicKeys:    publicKeys,
	}
	compTx, err := bws.syncClient.BtcRpcClient.BuildSignedTransaction(context.Background(), signedReq)
	if err != nil {
		log.Error("create un sign transaction fail", "err", err)
		return nil, err
	}
	log.Info("signed transaction data", "SignedTxData", compTx.SignedTxData)
	var retSignedTxn []*dal_wallet_go.ReturnSignedTransactions
	retSign := &dal_wallet_go.ReturnSignedTransactions{
		TransactionUuid: transactionId,
		SignedTx:        string(compTx.SignedTxData),
	}

	err = bws.db.Withdraws.UpdateWithdrawByGuuid(request.RequestId, transactionId, string(compTx.SignedTxData))
	if err != nil {
		log.Error("update withdraw fail", "err", err)
		return nil, err
	}

	retSignedTxn = append(retSignedTxn, retSign)
	resp.Msg = "create signed tx success"
	resp.Code = dal_wallet_go.ReturnCode_SUCCESS
	resp.ReturnSignTxn = retSignedTxn
	return resp, nil
}

func (bws *BusinessMiddleWireServices) SubmitWithdraw(ctx context.Context, request *dal_wallet_go.SubmitWithdrawRequest) (*dal_wallet_go.SubmitWithdrawResponse, error) {
	resp := &dal_wallet_go.SubmitWithdrawResponse{
		Code: dal_wallet_go.ReturnCode_ERROR,
		Msg:  "submit withdraw fail",
	}
	if request.ConsumerToken != ConsumerToken {
		resp.Msg = "consumer token is error"
		return resp, nil
	}
	var childTxList []database.ChildTxs
	txId := uuid.New()
	withdrawTimeStamp := uint64(time.Now().Unix())
	for _, withdraw := range request.WithdrawList {
		childTx := database.ChildTxs{
			GUID:        uuid.New(),
			Hash:        "0x0",
			TxId:        txId.String(),
			TxIndex:     big.NewInt(0),
			TxType:      "withdraw",
			FromAddress: "hotwallet",
			ToAddress:   withdraw.Address,
			Amount:      withdraw.Value,
			Timestamp:   withdrawTimeStamp,
		}
		childTxList = append(childTxList, childTx)
	}
	withdraw := &database.Withdraws{
		Guid:        uuid.New(),
		BlockHash:   "0x0",
		BlockNumber: big.NewInt(0),
		Hash:        "0x0",
		Fee:         big.NewInt(0),
		LockTime:    big.NewInt(0),
		Version:     "0x0",
		TxSignHex:   "0x0",
		Status:      database.TxStatusWaitSign,
		Timestamp:   withdrawTimeStamp,
	}
	if err := bws.db.Transaction(func(tx *database.DB) error {
		if len(childTxList) > 0 {
			if err := tx.ChildTxs.StoreChildTxs(request.RequestId, childTxList); err != nil {
				log.Error("store child txs fail", "err", err)
				return err
			}
		}
		if err := tx.Withdraws.StoreWithdraws(request.RequestId, withdraw); err != nil {
			log.Error("store child txs fail", "err", err)
			return err
		}
		return nil
	}); err != nil {
		log.Error("unable to persist withdraw tx batch", "err", err)
		return nil, err
	}
	return nil, nil
}
