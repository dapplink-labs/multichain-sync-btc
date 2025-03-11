package notifier

type NotifyRequest struct {
	Txn []Transaction `json:"txn"`
}

type Transaction struct {
	BlockHash    string `json:"block_hash"`
	BlockNumber  uint64 `json:"block_number"`
	Hash         string `json:"hash"`
	FromAddress  string `json:"from_address"`
	ToAddress    string `json:"to_address"`
	Value        string `json:"value"`
	Fee          string `json:"fee"`
	TxType       string `json:"tx_type"` // 0: 充值，1:提现；2:归集，3:热转冷；4:冷转热
	Confirms     uint8  `json:"confirms"`
	TokenAddress string `json:"token_address"`
	TokenId      string `json:"token_id"`
	TokenMeta    string `json:"token_meta"`
}

type NotifyResponse struct {
	Success bool `json:"success"`
}
