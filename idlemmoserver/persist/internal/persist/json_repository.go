package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/idle-server/common"
)

// JSONRepository JSON文件仓库
type JSONRepository struct {
	mu   sync.Mutex
	path string
}

// NewJSONRepository 创建JSON仓库
func NewJSONRepository(saveDir string) *JSONRepository {
	_ = os.MkdirAll(saveDir, 0755)
	return &JSONRepository{path: saveDir}
}

// filePath 获取文件路径
func (r *JSONRepository) filePath(playerID string) string {
	return filepath.Join(r.path, fmt.Sprintf("player_%s.json", playerID))
}

// SavePlayer 保存玩家数据
func (r *JSONRepository) SavePlayer(data *common.PlayerData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f := r.filePath(data.PlayerID)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal player data: %w", err)
	}
	return os.WriteFile(f, b, 0644)
}

// LoadPlayer 加载玩家数据
func (r *JSONRepository) LoadPlayer(playerID string) (*common.PlayerData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f := r.filePath(playerID)
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("load player file: %w", err)
	}
	var data common.PlayerData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("parse player json: %w", err)
	}
	return &data, nil
}

// PlayerExists 检查玩家是否存在
func (r *JSONRepository) PlayerExists(playerID string) bool {
	f := r.filePath(playerID)
	_, err := os.Stat(f)
	return !os.IsNotExist(err)
}

// DeletePlayer 删除玩家数据
func (r *JSONRepository) DeletePlayer(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f := r.filePath(playerID)
	err := os.Remove(f)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete player file: %w", err)
	}
	return nil
}

// Save 实现PlayerRepository接口的Save方法
func (r *JSONRepository) Save(playerID string, data *common.PlayerData) error {
	return r.SavePlayer(data)
}

// Load 实现PlayerRepository接口的Load方法
func (r *JSONRepository) Load(playerID string) (*common.PlayerData, error) {
	return r.LoadPlayer(playerID)
}

// Exists 实现PlayerRepository接口的Exists方法
func (r *JSONRepository) Exists(playerID string) bool {
	return r.PlayerExists(playerID)
}

// Delete 实现PlayerRepository接口的Delete方法
func (r *JSONRepository) Delete(playerID string) error {
	return r.DeletePlayer(playerID)
}
