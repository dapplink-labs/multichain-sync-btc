package signclient

type SignedTransaction struct {
	TxMessageHash string `json:"tx_message_hash"`
	TxHash        string `json:"tx_hash"`
	SignedTx      string `json:"signed_tx"`
}
