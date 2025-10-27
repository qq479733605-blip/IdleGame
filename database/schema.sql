-- Idle Server Database Schema
-- 修仙放置游戏数据库结构

-- 创建数据库
CREATE DATABASE IF NOT EXISTS idle_server CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE idle_server;

-- 用户表 - 存储用户认证信息
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    player_id VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_login TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE,
    INDEX idx_username (username),
    INDEX idx_player_id (player_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 玩家表 - 存储玩家游戏数据
CREATE TABLE IF NOT EXISTS players (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    player_id VARCHAR(64) NOT NULL UNIQUE,
    username VARCHAR(50) NOT NULL,
    level INT DEFAULT 1,
    exp BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_save_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_online BOOLEAN DEFAULT FALSE,
    -- JSON 字段存储灵活的游戏数据
    game_data JSON DEFAULT '{}',
    -- 预留字段用于将来的游戏功能
    total_playtime BIGINT DEFAULT 0,
    login_count INT DEFAULT 0,
    INDEX idx_player_id (player_id),
    INDEX idx_username (username),
    INDEX idx_level (level),
    INDEX idx_exp (exp),
    INDEX idx_is_online (is_online),
    INDEX idx_last_save_time (last_save_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 游戏进度表 - 为将来的游戏功能预留
CREATE TABLE IF NOT EXISTS game_progress (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    player_id VARCHAR(64) NOT NULL,
    progress_type VARCHAR(50) NOT NULL, -- 'sequence', 'achievement', 'quest', etc.
    progress_key VARCHAR(100) NOT NULL,
    progress_value JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_progress (player_id, progress_type, progress_key),
    INDEX idx_player_id (player_id),
    INDEX idx_progress_type (progress_type),
    FOREIGN KEY (player_id) REFERENCES players(player_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建示例用户（开发测试用）
-- 注意：这里的密码是 'password123' 的 bcrypt 哈希值
INSERT IGNORE INTO users (username, password_hash, player_id) VALUES
('admin', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'player_admin_001'),
('testuser', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'player_test_001');

-- 为示例用户创建对应的玩家记录
INSERT IGNORE INTO players (player_id, username, game_data) VALUES
('player_admin_001', 'admin', '{"rank": "admin", "privileges": ["all"]}'),
('player_test_001', 'testuser', '{"rank": "user", "tutorial_completed": true}');