package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"idlemmoserver/common"
)

type Repository interface {
	Save(snapshot common.PlayerSnapshot) error
	Load(playerID string) (common.PlayerSnapshot, bool, error)
}

type JSONRepository struct {
	mu   sync.Mutex
	path string
}

func NewJSONRepository(dir string) (Repository, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &JSONRepository{path: dir}, nil
}

func (r *JSONRepository) file(playerID string) string {
	return filepath.Join(r.path, fmt.Sprintf("player_%s.json", playerID))
}

func (r *JSONRepository) Save(snapshot common.PlayerSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.file(snapshot.PlayerID), data, 0o644)
}

func (r *JSONRepository) Load(playerID string) (common.PlayerSnapshot, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.file(playerID))
	if err != nil {
		if os.IsNotExist(err) {
			return common.PlayerSnapshot{}, false, nil
		}
		return common.PlayerSnapshot{}, false, err
	}
	var snapshot common.PlayerSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return common.PlayerSnapshot{}, false, err
	}
	return snapshot, true, nil
}
