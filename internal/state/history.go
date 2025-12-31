package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TransactionChange represents a single change within a transaction
type TransactionChange struct {
	Type       string `json:"type"`        // resource type (file, pkg)
	Name       string `json:"name"`        // resource name
	Target     string `json:"target"`      // actual system path or pkg name
	Action     string `json:"action"`      // installed, modified, removed
	BackupPath string `json:"backup_path"` // path to backup file (if any)
	PrevState  string `json:"prev_state"`  // state before change (absent, present)
}

// Transaction represents a complete application run
type Transaction struct {
	ID        string              `json:"id"`
	Timestamp string              `json:"timestamp"`
	Status    string              `json:"status"` // success, failed, reverted
	Changes   []TransactionChange `json:"changes"`
}

// HistoryManager manages the persistent history of transactions
type HistoryManager struct {
	HistoryFile string
}

func NewHistoryManager(baseDir string) *HistoryManager {
	if baseDir == "" {
		home, _ := os.UserHomeDir()
		baseDir = filepath.Join(home, ".monarch")
	}
	return &HistoryManager{
		HistoryFile: filepath.Join(baseDir, "history.json"),
	}
}

// AddTransaction appends a new transaction to the history
func (hm *HistoryManager) AddTransaction(tx Transaction) error {
	history, err := hm.LoadHistory()
	if err != nil {
		history = []Transaction{}
	}

	// Prepend for "newest first" or Append?
	// Usually logs are appended, but UI shows newest first.
	// Let's append to file, reverse in UI.
	history = append(history, tx)

	return hm.saveHistory(history)
}

// LoadHistory reads the history file
func (hm *HistoryManager) LoadHistory() ([]Transaction, error) {
	if _, err := os.Stat(hm.HistoryFile); os.IsNotExist(err) {
		return []Transaction{}, nil
	}

	data, err := os.ReadFile(hm.HistoryFile)
	if err != nil {
		return nil, err
	}

	var history []Transaction
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	return history, nil
}

// GetTransaction finds a transaction by ID
func (hm *HistoryManager) GetTransaction(id string) (*Transaction, error) {
	history, err := hm.LoadHistory()
	if err != nil {
		return nil, err
	}

	for _, tx := range history {
		if tx.ID == id {
			return &tx, nil
		}
	}
	return nil, fmt.Errorf("transaction not found: %s", id)
}

func (hm *HistoryManager) saveHistory(history []Transaction) error {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(hm.HistoryFile, data, 0644)
}

// GenerateID creates a simple unique ID
func GenerateID() string {
	return fmt.Sprintf("run-%s", time.Now().Format("20060102-150405"))
}
