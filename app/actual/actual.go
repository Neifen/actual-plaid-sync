package actual

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Actual struct {
	baseUrl    *url.URL
	accountUrl *url.URL
	AccountID  string
	apiKey     string
}

func NewActual() *Actual {
	envUrl := os.Getenv("ACTUAL_SERVER_URL")
	fileSyncId := os.Getenv("FILE_SYNC_ID")
	apiKey := os.Getenv("ACTUAL_HTTP_API_KEY")

	if envUrl == "" {
		log.Fatal("ACTUAL_SERVER_URL missing in .env file")
	}

	if fileSyncId == "" {
		log.Fatal("FILE_SYNC_ID missing in .env file")
	}

	if apiKey == "" {
		log.Fatal("ACTUAL_HTTP_API_KEY missing in .env file")
	}

	baseUrl, err := url.Parse(envUrl)
	baseUrl = baseUrl.JoinPath("v1", "budgets", fileSyncId)
	if err != nil {
		log.Fatalf("error parsing url %s: %v", envUrl, err)
	}

	a := &Actual{
		baseUrl: baseUrl,
		apiKey:  apiKey,
	}

	accName := os.Getenv("ACCOUNT_NAME")
	if accName == "" {
		log.Fatal("ACCOUNT_NAME missing in .env file")
	}

	accountID, err := a.accountId(accName)
	a.AccountID = accountID
	if err != nil {
		log.Fatalf("error parsing url %s: %v", envUrl, err)
	}

	a.accountUrl = baseUrl.JoinPath("accounts", accountID)
	return a
}

func (a *Actual) accountId(name string) (string, error) {
	queryUrl := a.baseUrl.JoinPath("id-by-name")

	q := queryUrl.Query()
	q.Set("type", "accounts")
	q.Set("name", name)
	queryUrl.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, queryUrl.String(), nil)
	if err != nil {
		return "", fmt.Errorf("reading account ID, create GET: %w", err)
	}

	req.Header.Add("x-api-key", a.apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("reading account ID, GET: %w", err)
	}
	defer res.Body.Close()

	var response struct {
		Data  string `json:"data"`
		Error string `json:"error"`
	}

	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("reading account ID, Decode: %w", err)
	}

	return response.Data, nil
}

func (a *Actual) FindLast(maxDays int) (*Transaction, *time.Time, error) {
	date := time.Now()
	for range maxDays {
		transactions, err := a.Transactions(date)
		if err != nil {
			return nil, nil, fmt.Errorf("find last: %w", err)
		}

		if len(transactions) > 0 {
			// return last transaction
			return &transactions[len(transactions)-1], &date, nil
		}

		// -1 day
		date = date.Add(time.Hour * -24)
	}

	return nil, nil, fmt.Errorf("could not find last transaction in the last %d days", maxDays)
}

type Transaction struct {
	ID         string `json:"id,omitempty"`
	Amount     int    `json:"amount"`
	Notes      string `json:"notes"`
	ImportedID string `json:"imported_id"`
	Date       string `json:"date"`
	Payee      string `json:"imported_payee"`
	Cleared    bool   `json:"cleared"`
	AccountID  string `json:"account"`
}

func (a *Actual) Transactions(date time.Time) ([]Transaction, error) {
	queryUrl := a.accountUrl.JoinPath("transactions")

	q := queryUrl.Query()
	q.Set("since_date", date.Format("2006-01-02"))
	queryUrl.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, queryUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("reading transactions, create GET: %w", err)
	}

	req.Header.Add("x-api-key", a.apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reading transactions: %w", err)
	}
	defer res.Body.Close()

	var response struct {
		Transactions []Transaction `json:"data"`
		Error        string        `json:"error"`
	}

	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("reading transactions: %w", err)
	}

	return response.Transactions, nil
}

func (a *Actual) ImportTransactions(transactions []Transaction) error {

	body := struct {
		Transactions    []Transaction `json:"transactions"`
		DefaultCleared  bool          `json:"defaultCleared"`
		DryRun          bool          `json:"dryRun"`
		ReimportDeleted bool          `json:"reimportDeleted"`
	}{
		Transactions:    transactions,
		DefaultCleared:  true,
		DryRun:          true, // change to end tests
		ReimportDeleted: false,
	}

	queryUrl := a.accountUrl.JoinPath("transactions", "import")

	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("importing transactions, marshall body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, queryUrl.String(), bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("importing transactions, create POST: %w", err)
	}

	req.Header.Add("x-api-key", a.apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("importing transactions, sending POST: %w", err)
	}
	defer res.Body.Close()

	var response struct {
		Data struct {
			Added   []string
			Updated []string
		} `json:"data"`
		Error string `json:"error"`
	}

	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		return fmt.Errorf("reading transactions: %w", err)
	}

	return nil
}

func (a *Actual) RemoveTransactions(transactionIds []string) error {

	body := struct {
		TransactionIds []string `json:"transactionIds"`
	}{
		TransactionIds: transactionIds,
	}

	queryUrl := a.accountUrl.JoinPath("transactions", "batch")

	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("deleting transactions, marshall body: %w", err)
	}

	req, err := http.NewRequest(http.MethodDelete, queryUrl.String(), bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("deleting transactions, create DELETE: %w", err)
	}

	req.Header.Add("x-api-key", a.apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("deleting transactions, sending DELETE: %w", err)
	}
	defer res.Body.Close()

	var response struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		return fmt.Errorf("deleting transactions: %w", err)
	}

	return nil
}
