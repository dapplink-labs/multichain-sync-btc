package dynamic

import (
	"fmt"

	"github.com/dapplink-labs/multichain-sync-btc/database"
)

func CreateTableFromTemplate(requestId string, db *database.DB) {
	createAddresses(requestId, db)
	createVins(requestId, db)
	createVouts(requestId, db)
	createBalances(requestId, db)
	createDeposits(requestId, db)
	createTransactions(requestId, db)
	createWithdraws(requestId, db)
	createInternals(requestId, db)
	createChildTxn(requestId, db)
}

func createAddresses(requestId string, db *database.DB) {
	tableName := "addresses"
	tableNameByChainId := fmt.Sprintf("addresses_%s", requestId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}

func createVins(requestId string, db *database.DB) {
	tableName := "vins"
	tableNameByChainId := fmt.Sprintf("vins_%s", requestId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}

func createVouts(requestId string, db *database.DB) {
	tableName := "vouts"
	tableNameByChainId := fmt.Sprintf("vouts_%s", requestId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}

func createBalances(requestId string, db *database.DB) {
	tableName := "balances"
	tableNameByChainId := fmt.Sprintf("balances_%s", requestId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}

func createDeposits(requestId string, db *database.DB) {
	tableName := "deposits"
	tableNameByChainId := fmt.Sprintf("deposits_%s", requestId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}

func createTransactions(requestId string, db *database.DB) {
	tableName := "transactions"
	tableNameByChainId := fmt.Sprintf("transactions_%s", requestId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}

func createWithdraws(requestId string, db *database.DB) {
	tableName := "withdraws"
	tableNameByChainId := fmt.Sprintf("withdraws_%s", requestId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}

func createInternals(requestId string, db *database.DB) {
	tableName := "internals"
	tableNameByChainId := fmt.Sprintf("internals_%s", requestId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}

func createChildTxn(requestId string, db *database.DB) {
	tableName := "child_txs"
	tableNameByChainId := fmt.Sprintf("child_txs_%s", requestId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}
