package database

type TxStatus uint8

const (
	TxStatusUnsigned    TxStatus = 0 // 交易未签名
	TxStatusSigned      TxStatus = 1 // 交易交易已签名
	TxStatusBroadcasted TxStatus = 2 // 交易已经发送到区块链网络
	TxStatusWalletDone  TxStatus = 3 // 交易在钱包层已完成
	TxStatusNotified    TxStatus = 4 // 交易已通知业务
	TxStatusSuccess     TxStatus = 5 // 交易成功
)
