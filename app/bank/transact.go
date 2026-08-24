package bank

import (
	"fmt"
	"log"
	"time"

	"github.com/plaid/plaid-go/v46/plaid"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("config")
}

func (b *Bank) ResetTransactionsToday() plaid.TransactionsSyncResponse {
	cursor := "now" // start today
	return b.syncTransactions(nil, cursor)
}

func (b *Bank) ResetTransactions(cutoffDate *time.Time) plaid.TransactionsSyncResponse {
	cursor := "" // full reset
	b.saveCursor(cursor)
	return b.syncTransactions(cutoffDate, cursor)
}

func (b *Bank) Transactions() plaid.TransactionsSyncResponse {
	cursor := b.getCursor()
	return b.syncTransactions(nil, cursor)
}

func (b *Bank) getCursor() string {
	return b.SyncState.Cursor
}

func (b *Bank) saveCursor(c string) error {
	b.SyncState.Cursor = c
	b.SyncState.LastSyncedAt = time.Now()

	stateFile, err := LoadState()
	if err != nil {
		return fmt.Errorf("saving cursor: %w", err)
	}

	stateFile[b.name] = b.SyncState
	if err = SaveState(stateFile); err != nil {
		return fmt.Errorf("saving cursor: %w", err)
	}

	return nil
}

func (b *Bank) syncTransactions(startDate *time.Time, cursor string) plaid.TransactionsSyncResponse {
	syncReq := plaid.NewTransactionsSyncRequest(b.tokenAccess.accessToken())
	syncReq.SetCursor(cursor)
	syncRes, _, err := b.client.PlaidApi.TransactionsSync(b.ctx).TransactionsSyncRequest(*syncReq).Execute()
	if err != nil {
		plaidErr, plaidErrParseErr := plaid.ToPlaidError(err)
		if plaidErrParseErr == nil {
			fmt.Printf("Plaid error: %s | %s | %s\n",
				plaidErr.ErrorCode, plaidErr.ErrorType, plaidErr.ErrorMessage)
		}
		log.Fatalf("transactions sync failed: %v", err)
	}

	fmt.Printf("Added: %d, Modified: %d, Removed: %d\n",
		len(syncRes.GetAdded()),
		len(syncRes.GetModified()),
		len(syncRes.GetRemoved()),
	)

	syncRes.Added = filterTransactions(startDate, syncRes.Added)
	syncRes.Modified = filterTransactions(startDate, syncRes.Modified)

	b.saveCursor(syncRes.NextCursor)

	return syncRes
}

func filterTransactions(startDate *time.Time, transactions []plaid.Transaction) []plaid.Transaction {
	if startDate == nil || startDate.IsZero() {
		return transactions
	}

	var filtered []plaid.Transaction
	for _, txn := range transactions {
		txDate, err := time.Parse("2006-01-02", txn.GetDate())
		if err != nil {
			fmt.Print("Error parsing date: ", txn.GetDate())
			continue
		}

		if startDate.After(txDate) {
			continue
		}

		filtered = append(filtered, txn)
	}

	return filtered
}
