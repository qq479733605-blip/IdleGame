package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// DefaultRedisConfig 默认Redis配置
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}
}

// Redis Redis连接管理
type Redis struct {
	client *redis.Client
	config *RedisConfig
}

// NewRedis 创建Redis连接
func NewRedis(config *RedisConfig) (*Redis, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Connected to Redis: %s:%d, DB: %d", config.Host, config.Port, config.DB)

	redis := &Redis{
		client: rdb,
		config: config,
	}

	return redis, nil
}

// GetClient 获取Redis客户端
func (r *Redis) GetClient() *redis.Client {
	return r.client
}

// Close 关闭Redis连接
func (r *Redis) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// Ping 检查Redis连接
func (r *Redis) Ping(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	_, err := r.client.Ping(ctx).Result()
	return err
}

// IsHealthy 检查Redis健康状态
func (r *Redis) IsHealthy() bool {
	if r.client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Redis health check failed: %v", err)
		return false
	}

	return true
}

// ============ 缓存操作方法 ============

// SetUserSession 设置用户会话缓存
func (r *Redis) SetUserSession(ctx context.Context, playerID, token string, expiration time.Duration) error {
	key := fmt.Sprintf("session:%s", playerID)
	return r.client.Set(ctx, key, token, expiration).Err()
}

// GetUserSession 获取用户会话缓存
func (r *Redis) GetUserSession(ctx context.Context, playerID string) (string, error) {
	key := fmt.Sprintf("session:%s", playerID)
	return r.client.Get(ctx, key).Result()
}

// DeleteUserSession 删除用户会话缓存
func (r *Redis) DeleteUserSession(ctx context.Context, playerID string) error {
	key := fmt.Sprintf("session:%s", playerID)
	return r.client.Del(ctx, key).Err()
}

// SetPlayerData 设置玩家数据缓存
func (r *Redis) SetPlayerData(ctx context.Context, playerID string, data interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("player:%s", playerID)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal player data: %w", err)
	}
	return r.client.Set(ctx, key, jsonData, expiration).Err()
}

// GetPlayerData 获取玩家数据缓存
func (r *Redis) GetPlayerData(ctx context.Context, playerID string, dest interface{}) error {
	key := fmt.Sprintf("player:%s", playerID)
	jsonData, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonData), dest)
}

// DeletePlayerData 删除玩家数据缓存
func (r *Redis) DeletePlayerData(ctx context.Context, playerID string) error {
	key := fmt.Sprintf("player:%s", playerID)
	return r.client.Del(ctx, key).Err()
}

// SetOnlinePlayer 设置在线玩家
func (r *Redis) SetOnlinePlayer(ctx context.Context, playerID string, expiration time.Duration) error {
	key := "online_players"
	return r.client.SAdd(ctx, key, playerID).Err()
}

// RemoveOnlinePlayer 移除在线玩家
func (r *Redis) RemoveOnlinePlayer(ctx context.Context, playerID string) error {
	key := "online_players"
	return r.client.SRem(ctx, key, playerID).Err()
}

// GetOnlinePlayers 获取在线玩家列表
func (r *Redis) GetOnlinePlayers(ctx context.Context) ([]string, error) {
	key := "online_players"
	return r.client.SMembers(ctx, key).Result()
}

// IsPlayerOnline 检查玩家是否在线
func (r *Redis) IsPlayerOnline(ctx context.Context, playerID string) (bool, error) {
	key := "online_players"
	count, err := r.client.SIsMember(ctx, key, playerID).Result()
	return count, err
}

// SetRanking 设置排行榜数据
func (r *Redis) SetRanking(ctx context.Context, rankingType string, playerID string, score float64) error {
	key := fmt.Sprintf("ranking:%s", rankingType)
	return r.client.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: playerID,
	}).Err()
}

// GetRanking 获取排行榜
func (r *Redis) GetRanking(ctx context.Context, rankingType string, start, stop int64) ([]redis.Z, error) {
	key := fmt.Sprintf("ranking:%s", rankingType)
	return r.client.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

// GetPlayerRank 获取玩家排名
func (r *Redis) GetPlayerRank(ctx context.Context, rankingType string, playerID string) (int64, error) {
	key := fmt.Sprintf("ranking:%s", rankingType)
	return r.client.ZRevRank(ctx, key, playerID).Result()
}
