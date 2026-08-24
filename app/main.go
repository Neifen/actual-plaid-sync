package main

import (
	"actual-plaid-sync/app/actual"
	"actual-plaid-sync/app/bank"
	"log"

	"github.com/joho/godotenv"
	"github.com/plaid/plaid-go/v46/plaid"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on real Envionmental variables")
	}

	act := actual.NewActual()
	paidClient := bank.PlaidClient()
	bank := bank.Sandbox(paidClient)

	var transactions plaid.TransactionsSyncResponse
	if bank.HasCursor() {
		transactions = bank.Transactions()
	} else {
		_, date, err := act.FindLast(60)
		if err != nil {
			log.Printf("error with actual: %v\n", err)
		}

		transactions = bank.ResetTransactions(date)
	}

	added := actual.PlaidToActualTransaction(transactions.Added, act.AccountID)
	modified := actual.PlaidToActualTransaction(transactions.Modified, act.AccountID)
	removed := actual.PlaidToActualRemovedTransaction(transactions.Removed)

	if err := act.ImportTransactions(added); err != nil {
		log.Fatalf("error importing transactions: %v", err)
	}
	if err := act.ImportTransactions(modified); err != nil {
		log.Fatalf("error importing transactions: %v", err)
	}
	if err := act.RemoveTransactions(removed); err != nil {
		log.Fatalf("error removing transactions: %v", err)
	}
}
