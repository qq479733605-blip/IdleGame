package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"idlemmoserver/internal/common"
)

type JSONRepo struct {
	mu   sync.Mutex
	path string
}

func NewJSONRepo(saveDir string) *JSONRepo {
	_ = os.MkdirAll(saveDir, 0755)
	return &JSONRepo{path: saveDir}
}

func (r *JSONRepo) filePath(playerID string) string {
	return filepath.Join(r.path, fmt.Sprintf("player_%s.json", playerID))
}

func (r *JSONRepo) SavePlayer(data *common.PlayerSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f := r.filePath(data.PlayerID)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f, b, 0644)
}

func (r *JSONRepo) LoadPlayer(playerID string) (*common.PlayerSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f := r.filePath(playerID)
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("load failed: %w", err)
	}
	var d common.PlayerSnapshot
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return &d, nil
}
