package bank

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/plaid/plaid-go/v46/plaid"
)

type Config struct {
}

func PlaidClient() *plaid.APIClient {
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", os.Getenv("PLAID_CLIENT_ID"))
	configuration.AddDefaultHeader("PLAID-SECRET", os.Getenv("PLAID_SECRET"))
	configuration.UseEnvironment(plaid.Sandbox) // or plaid.Production
	return plaid.NewAPIClient(configuration)
}

type Bank struct {
	ctx    context.Context
	client *plaid.APIClient

	SyncState   SyncState
	name        string
	tokenAccess tokenerAccesser
}

const stateFilePath = "sync-state.json"

type SyncStateFile map[string]SyncState
type SyncState struct {
	Cursor       string    `json:"cursor"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}

func (b *Bank) HasCursor() bool {
	return b.SyncState.Cursor != ""
}

func LoadState() (SyncStateFile, error) {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return SyncStateFile{}, nil // its fine, there is just no state saved yet
		}
		return SyncStateFile{}, fmt.Errorf("loading syncState file: %w", err)
	}

	var state SyncStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return SyncStateFile{}, fmt.Errorf("loading syncState file: %w", err)
	}

	return state, nil
}

func SaveState(state SyncStateFile) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("saving SyncState: %w", err)
	}

	err = os.WriteFile(stateFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("saving SyncState: %w", err)
	}
	return nil
}

func Bluevine(client *plaid.APIClient) *Bank {
	name := "Bluevine"
	cursorState, err := LoadState()
	if err != nil {
		log.Fatalf("could not load syncstate: %v", err)
	}

	return &Bank{
		ctx:         context.Background(),
		client:      client,
		name:        name,
		SyncState:   cursorState[name],
		tokenAccess: bluevineAccesser{},
	}
}

func Sandbox(client *plaid.APIClient) *Bank {
	name := "Sandbox"
	cursorState, err := LoadState()
	if err != nil {
		log.Fatalf("could not load syncstate: %v", err)
	}

	return &Bank{
		ctx:         context.Background(),
		client:      client,
		name:        name,
		SyncState:   cursorState[name],
		tokenAccess: sandboxAccesser{},
	}
}
