package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/idle-server/common"
	"gorm.io/gorm"
)

// GORMPlayerRepository GORM玩家仓库
type GORMPlayerRepository struct {
	db    *gorm.DB
	redis *Redis
}

// NewGORMPlayerRepository 创建GORM玩家仓库
func NewGORMPlayerRepository(db *gorm.DB, redis *Redis) *GORMPlayerRepository {
	return &GORMPlayerRepository{
		db:    db,
		redis: redis,
	}
}

// SavePlayerData 保存玩家数据
func (r *GORMPlayerRepository) SavePlayerData(ctx context.Context, playerData *common.PlayerData) error {
	// 序列化游戏数据
	gameDataJSON, err := json.Marshal(playerData)
	if err != nil {
		return fmt.Errorf("failed to marshal player data: %w", err)
	}

	// 更新数据库
	result := r.db.WithContext(ctx).Model(&Player{}).
		Where("player_id = ?", playerData.PlayerID).
		Updates(map[string]interface{}{
			"last_save_time": time.Now(),
			"game_data":      string(gameDataJSON),
			"updated_at":     time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to save player data: %w", result.Error)
	}

	// 更新缓存
	if r.redis != nil {
		if err := r.cachePlayerData(ctx, playerData); err != nil {
			log.Printf("Failed to cache player data: %v", err)
		}
	}

	log.Printf("Player data saved: %s", playerData.PlayerID)
	return nil
}

// LoadPlayerData 加载玩家数据
func (r *GORMPlayerRepository) LoadPlayerData(ctx context.Context, playerID string) (*common.PlayerData, error) {
	// 先从缓存获取
	if r.redis != nil {
		if playerData := r.getCachedPlayerData(ctx, playerID); playerData != nil {
			return playerData, nil
		}
	}

	// 从数据库获取
	var player Player
	err := r.db.WithContext(ctx).Where("player_id = ?", playerID).First(&player).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("player %s not found", playerID)
		}
		return nil, fmt.Errorf("failed to load player data: %w", err)
	}

	// 转换为公共数据结构
	playerData := player.ToCommonPlayerData()

	// 解析游戏数据中的额外字段
	if player.GameData != "" && player.GameData != "{}" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(player.GameData), &extraData); err == nil {
			// 合并额外数据到 playerData
			if level, ok := extraData["level"].(float64); ok {
				playerData.Level = int(level)
			}
			if exp, ok := extraData["exp"].(float64); ok {
				playerData.Exp = int64(exp)
			}
		}
	}

	// 缓存结果
	if r.redis != nil {
		if err := r.cachePlayerData(ctx, playerData); err != nil {
			log.Printf("Failed to cache player data: %v", err)
		}
	}

	return playerData, nil
}

// PlayerExists 检查玩家是否存在
func (r *GORMPlayerRepository) PlayerExists(ctx context.Context, playerID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Player{}).Where("player_id = ?", playerID).Count(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check player existence: %w", err)
	}
	return count > 0, nil
}

// UpdatePlayerStatus 更新玩家状态（在线状态等）
func (r *GORMPlayerRepository) UpdatePlayerStatus(ctx context.Context, playerID string, isOnline bool) error {
	result := r.db.WithContext(ctx).Model(&Player{}).
		Where("player_id = ?", playerID).
		Updates(map[string]interface{}{
			"is_online":  isOnline,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update player status: %w", result.Error)
	}

	// 更新Redis在线状态
	if r.redis != nil {
		if isOnline {
			r.redis.SetOnlinePlayer(ctx, playerID, 24*time.Hour)
		} else {
			r.redis.RemoveOnlinePlayer(ctx, playerID)
		}
	}

	return nil
}

// UpdatePlayerLevel 更新玩家等级
func (r *GORMPlayerRepository) UpdatePlayerLevel(ctx context.Context, playerID string, level int, exp int64) error {
	// 获取当前玩家数据以更新游戏数据
	playerData, err := r.LoadPlayerData(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to load player data for level update: %w", err)
	}

	// 更新等级和经验
	playerData.Level = level
	playerData.Exp = exp

	// 保存更新后的数据
	return r.SavePlayerData(ctx, playerData)
}

// IncrementLoginCount 增加登录次数
func (r *GORMPlayerRepository) IncrementLoginCount(ctx context.Context, playerID string) error {
	result := r.db.WithContext(ctx).Model(&Player{}).
		Where("player_id = ?", playerID).
		UpdateColumn("login_count", gorm.Expr("login_count + 1"))

	if result.Error != nil {
		return fmt.Errorf("failed to increment login count: %w", result.Error)
	}

	return nil
}

// GetOnlinePlayers 获取在线玩家列表
func (r *GORMPlayerRepository) GetOnlinePlayers(ctx context.Context) ([]string, error) {
	// 先从Redis获取
	if r.redis != nil {
		if onlinePlayers, err := r.redis.GetOnlinePlayers(ctx); err == nil && len(onlinePlayers) > 0 {
			return onlinePlayers, nil
		}
	}

	// 从数据库获取
	var playerIDs []string
	err := r.db.WithContext(ctx).Model(&Player{}).
		Where("is_online = ?", true).
		Pluck("player_id", &playerIDs).Error

	return playerIDs, err
}

// UpdatePlaytime 更新玩家游戏时间
func (r *GORMPlayerRepository) UpdatePlaytime(ctx context.Context, playerID string, additionalTime int64) error {
	result := r.db.WithContext(ctx).Model(&Player{}).
		Where("player_id = ?", playerID).
		UpdateColumn("total_playtime", gorm.Expr("total_playtime + ?", additionalTime))

	if result.Error != nil {
		return fmt.Errorf("failed to update playtime: %w", result.Error)
	}
	return nil
}

// GetPlayerStats 获取玩家统计信息
func (r *GORMPlayerRepository) GetPlayerStats(ctx context.Context, playerID string) (map[string]interface{}, error) {
	var player Player
	err := r.db.WithContext(ctx).Select(
		"level", "exp", "total_playtime", "login_count", "created_at", "last_save_time",
	).Where("player_id = ?", playerID).First(&player)

	if err != nil {
		return nil, fmt.Errorf("failed to get player stats: %w", err)
	}

	stats := map[string]interface{}{
		"level":               player.Level,
		"exp":                 player.Exp,
		"playtime":            player.TotalPlaytime,
		"login_count":         player.LoginCount,
		"created_at":          player.CreatedAt,
		"last_save":           player.LastSaveTime,
		"days_since_creation": int(time.Since(player.CreatedAt).Hours() / 24),
	}

	return stats, nil
}

// GetTopPlayersByLevel 获取等级排行榜
func (r *GORMPlayerRepository) GetTopPlayersByLevel(ctx context.Context, limit int) ([]common.PlayerRanking, error) {
	var players []Player
	err := r.db.WithContext(ctx).
		Select("player_id", "username", "level", "exp").
		Order("level DESC, exp DESC").
		Limit(limit).
		Find(&players).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get top players: %w", err)
	}

	var rankings []common.PlayerRanking
	for i, player := range players {
		rankings = append(rankings, common.PlayerRanking{
			Rank:     i + 1,
			PlayerID: player.PlayerID,
			Username: player.Username,
			Level:    player.Level,
			Exp:      player.Exp,
		})
	}

	return rankings, nil
}

// UpdatePlayerRanking 更新排行榜数据
func (r *GORMPlayerRepository) UpdatePlayerRanking(ctx context.Context, rankingType string, playerID string, score float64) error {
	if r.redis != nil {
		return r.redis.SetRanking(ctx, rankingType, playerID, score)
	}
	return nil
}

// GetPlayerRanking 获取玩家排名
func (r *GORMPlayerRepository) GetPlayerRanking(ctx context.Context, rankingType string, playerID string) (int64, error) {
	if r.redis != nil {
		return r.redis.GetPlayerRank(ctx, rankingType, playerID)
	}
	return 0, nil
}

// GetRanking 获取排行榜
func (r *GORMPlayerRepository) GetRanking(ctx context.Context, rankingType string, start, stop int64) ([]redis.Z, error) {
	if r.redis != nil {
		return r.redis.GetRanking(ctx, rankingType, start, stop)
	}
	return nil, fmt.Errorf("redis not available")
}

// DeletePlayer 删除玩家数据（谨慎使用）
func (r *GORMPlayerRepository) DeletePlayer(ctx context.Context, playerID string) error {
	// 开启事务
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除游戏进度
	if err := tx.Where("player_id = ?", playerID).Delete(&GameProgress{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete game progress: %w", err)
	}

	// 删除玩家记录
	if err := tx.Where("player_id = ?", playerID).Delete(&Player{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete player: %w", err)
	}

	// 删除用户记录
	if err := tx.Where("player_id = ?", playerID).Delete(&User{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 清除缓存
	if r.redis != nil {
		r.redis.DeletePlayerData(ctx, playerID)
		r.redis.RemoveOnlinePlayer(ctx, playerID)
	}

	log.Printf("Player deleted: %s", playerID)
	return nil
}

// BatchUpdateOfflinePlayers 批量更新离线玩家
func (r *GORMPlayerRepository) BatchUpdateOfflinePlayers(ctx context.Context, timeout time.Duration) error {
	cutoff := time.Now().Add(-timeout)

	result := r.db.WithContext(ctx).Model(&Player{}).
		Where("is_online = ? AND last_save_time < ?", true, cutoff).
		Update("is_online", false)

	if result.Error != nil {
		return fmt.Errorf("failed to batch update offline players: %w", result.Error)
	}

	log.Printf("Updated %d offline players", result.RowsAffected)

	// 可以在这里触发Redis在线玩家列表的清理
	if r.redis != nil {
		// 可以考虑定期清理Redis中的在线状态
	}

	return nil
}

// ============ 私有辅助方法 ============

// cachePlayerData 缓存玩家数据
func (r *GORMPlayerRepository) cachePlayerData(ctx context.Context, playerData *common.PlayerData) error {
	if r.redis == nil {
		return nil
	}

	// 缓存基础数据
	if err := r.redis.SetPlayerData(ctx, playerData.PlayerID, playerData, 30*time.Minute); err != nil {
		return err
	}

	return nil
}

// getCachedPlayerData 从缓存获取玩家数据
func (r *GORMPlayerRepository) getCachedPlayerData(ctx context.Context, playerID string) *common.PlayerData {
	if r.redis == nil {
		return nil
	}

	var playerData common.PlayerData
	if err := r.redis.GetPlayerData(ctx, playerID, &playerData); err != nil {
		return nil
	}

	return &playerData
}
