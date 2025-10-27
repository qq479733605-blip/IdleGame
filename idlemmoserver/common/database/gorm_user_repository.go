package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/idle-server/common"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// GORMUserRepository GORM用户仓库
type GORMUserRepository struct {
	db    *gorm.DB
	redis *Redis
}

// NewGORMUserRepository 创建GORM用户仓库
func NewGORMUserRepository(db *gorm.DB, redis *Redis) *GORMUserRepository {
	return &GORMUserRepository{
		db:    db,
		redis: redis,
	}
}

// CreateUser 创建新用户
func (r *GORMUserRepository) CreateUser(ctx context.Context, username, password string) (*common.UserData, error) {
	// 检查用户是否已存在
	exists, err := r.UserExists(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user %s already exists", username)
	}

	// 生成密码哈希
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 生成PlayerID
	playerID, err := r.generatePlayerID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate player ID: %w", err)
	}

	// 开始事务
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建用户记录
	user := &User{
		Username:     username,
		PasswordHash: string(passwordHash),
		PlayerID:     playerID,
		IsActive:     true,
	}

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 创建玩家记录
	player := &Player{
		PlayerID:      playerID,
		Username:      username,
		GameData:      "{}",
		TotalPlaytime: 0,
		LoginCount:    0,
	}

	if err := tx.Create(player).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create player record: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 创建用户数据对象
	userData := &common.UserData{
		Username:  username,
		Password:  string(passwordHash), // Include hashed password for authentication
		PlayerID:  playerID,
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}

	// 缓存用户数据
	if r.redis != nil {
		if err := r.cacheUser(ctx, userData); err != nil {
			log.Printf("Failed to cache user data: %v", err)
		}
	}

	log.Printf("User created successfully: %s (PlayerID: %s)", username, playerID)
	return userData, nil
}

// GetUserByUsername 根据用户名获取用户信息
func (r *GORMUserRepository) GetUserByUsername(ctx context.Context, username string) (*common.UserData, error) {
	// 先从缓存获取
	if r.redis != nil {
		if userData := r.getCachedUser(ctx, username); userData != nil {
			return userData, nil
		}
	}

	// 从数据库获取
	var user User
	err := r.db.WithContext(ctx).Preload("Player").Where("username = ? AND is_active = ?", username, true).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user %s not found", username)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userData := user.ToUserData()

	// 缓存结果
	if r.redis != nil {
		if err := r.cacheUser(ctx, userData); err != nil {
			log.Printf("Failed to cache user data: %v", err)
		}
	}

	return userData, nil
}

// GetUserByPlayerID 根据PlayerID获取用户信息
func (r *GORMUserRepository) GetUserByPlayerID(ctx context.Context, playerID string) (*common.UserData, error) {
	// 先从缓存获取
	if r.redis != nil {
		cacheKey := fmt.Sprintf("user_by_player:%s", playerID)
		var userData common.UserData
		if err := r.redis.GetPlayerData(ctx, cacheKey, &userData); err == nil {
			return &userData, nil
		}
	}

	// 从数据库获取
	var user User
	err := r.db.WithContext(ctx).Where("player_id = ? AND is_active = ?", playerID, true).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with playerID %s not found", playerID)
		}
		return nil, fmt.Errorf("failed to get user by playerID: %w", err)
	}

	userData := user.ToUserData()

	// 缓存结果
	if r.redis != nil {
		cacheKey := fmt.Sprintf("user_by_player:%s", playerID)
		if err := r.redis.SetPlayerData(ctx, cacheKey, &userData, 30*time.Minute); err != nil {
			log.Printf("Failed to cache user data: %v", err)
		}
	}

	return userData, nil
}

// UserExists 检查用户是否存在
func (r *GORMUserRepository) UserExists(ctx context.Context, username string) (bool, error) {
	// 先从缓存获取
	if r.redis != nil {
		cacheKey := fmt.Sprintf("user_exists:%s", username)
		if exists, err := r.redis.GetClient().Get(ctx, cacheKey).Bool(); err == nil {
			return exists, nil
		}
	}

	var count int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("username = ? AND is_active = ?", username, true).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	exists := count > 0

	// 缓存结果
	if r.redis != nil {
		cacheKey := fmt.Sprintf("user_exists:%s", username)
		r.redis.GetClient().Set(ctx, cacheKey, exists, 5*time.Minute)
	}

	return exists, nil
}

// AuthenticateUser 验证用户密码
func (r *GORMUserRepository) AuthenticateUser(ctx context.Context, username, password string) (*common.UserData, error) {
	// 获取用户信息
	userData, err := r.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// 从数据库获取密码哈希
	var user User
	err = r.db.WithContext(ctx).Select("password_hash").Where("username = ? AND is_active = ?", username, true).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get password hash: %w", err)
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	// 更新最后登录时间
	if err := r.db.Model(&User{}).Where("username = ?", username).Update("last_login", time.Now()).Error; err != nil {
		log.Printf("Failed to update last login time: %v", err)
	}

	// 清除相关缓存
	if r.redis != nil {
		r.redis.DeletePlayerData(ctx, fmt.Sprintf("user:%s", username))
		r.redis.GetClient().Del(ctx, fmt.Sprintf("user_exists:%s", username))
	}

	log.Printf("User authenticated successfully: %s", username)
	return userData, nil
}

// UpdateLastLogin 更新最后登录时间
func (r *GORMUserRepository) UpdateLastLogin(ctx context.Context, playerID string) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("player_id = ?", playerID).Update("last_login", time.Now()).Error
}

// DeleteUser 删除用户（软删除，设置为非活跃）
func (r *GORMUserRepository) DeleteUser(ctx context.Context, username string) error {
	result := r.db.WithContext(ctx).Model(&User{}).Where("username = ?", username).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to deactivate user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user %s not found", username)
	}

	// 清除缓存
	if r.redis != nil {
		r.redis.DeletePlayerData(ctx, fmt.Sprintf("user:%s", username))
		r.redis.GetClient().Del(ctx, fmt.Sprintf("user_exists:%s", username))
	}

	log.Printf("User deactivated: %s", username)
	return nil
}

// GetUsersByLevel 根据等级获取用户列表
func (r *GORMUserRepository) GetUsersByLevel(ctx context.Context, minLevel, maxLevel int, limit int) ([]*common.UserData, error) {
	var players []Player
	err := r.db.WithContext(ctx).
		Where("level BETWEEN ? AND ?", minLevel, maxLevel).
		Order("level DESC, exp DESC").
		Limit(limit).
		Find(&players).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get users by level: %w", err)
	}

	var users []*common.UserData
	for _, player := range players {
		// 手动查询用户数据
		var user User
		err := r.db.WithContext(ctx).Where("player_id = ? AND is_active = ?", player.PlayerID, true).First(&user).Error
		if err != nil {
			continue // 跳过找不到用户的玩家
		}

		userData := user.ToUserData()
		userData.Level = player.Level
		userData.Exp = player.Exp
		users = append(users, userData)
	}

	return users, nil
}

// ============ 私有辅助方法 ============

// generatePlayerID 生成新的PlayerID
func (r *GORMUserRepository) generatePlayerID() (string, error) {
	// 使用数据库的自增ID来生成唯一的PlayerID
	// 这里可以使用 UUID 或其他方法
	playerID := fmt.Sprintf("player_%d", time.Now().UnixNano())
	return playerID, nil
}

// cacheUser 缓存用户数据
func (r *GORMUserRepository) cacheUser(ctx context.Context, userData *common.UserData) error {
	if r.redis == nil {
		return nil
	}

	// 缓存用户数据
	if err := r.redis.SetPlayerData(ctx, fmt.Sprintf("user:%s", userData.Username), userData, 30*time.Minute); err != nil {
		return err
	}

	// 缓存存在性标记
	if err := r.redis.GetClient().Set(ctx, fmt.Sprintf("user_exists:%s", userData.Username), true, 5*time.Minute).Err(); err != nil {
		return err
	}

	return nil
}

// getCachedUser 从缓存获取用户数据
func (r *GORMUserRepository) getCachedUser(ctx context.Context, username string) *common.UserData {
	if r.redis == nil {
		return nil
	}

	var userData common.UserData
	cacheKey := fmt.Sprintf("user:%s", username)
	if err := r.redis.GetPlayerData(ctx, cacheKey, &userData); err != nil {
		return nil
	}

	return &userData
}
