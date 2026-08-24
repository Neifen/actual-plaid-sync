package actual

import "github.com/plaid/plaid-go/v46/plaid"

func PlaidToActualTransaction(plaid []plaid.Transaction, accountID string) []Transaction {
	actual := make([]Transaction, len(plaid))

	for i := range plaid {
		p := plaid[i]

		t := Transaction{
			AccountID:  accountID,
			Amount:     -int(p.GetAmount() * 100), // actual is weird like that
			Date:       p.GetDate(),
			Payee:      p.GetName(),
			Notes:      p.GetOriginalDescription(),
			Cleared:    !p.GetPending(),
			ImportedID: p.GetTransactionId(),
		}

		actual[i] = t
	}

	return actual
}

func PlaidToActualRemovedTransaction(plaid []plaid.RemovedTransaction) []string {
	actual := make([]string, len(plaid))

	for i := range plaid {
		p := plaid[i]

		actual[i] = p.GetTransactionId()
	}

	return actual
}
