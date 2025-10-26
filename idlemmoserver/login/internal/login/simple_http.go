package login

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/idle-server/common"
)

// SimpleHTTPHandler 简单的HTTP处理器，不使用Actor系统
type SimpleHTTPHandler struct {
	userRepo common.UserRepository
}

func NewSimpleHTTPHandler(userRepo common.UserRepository) *SimpleHTTPHandler {
	return &SimpleHTTPHandler{
		userRepo: userRepo,
	}
}

// HandleSimpleLogin 简单登录处理
func (h *SimpleHTTPHandler) HandleSimpleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 验证用户
	user, err := h.userRepo.GetUser(req.Username)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "User not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证密码 (这里应该有密码哈希验证)
	if user.Password != req.Password {
		response := map[string]interface{}{
			"success": false,
			"error":   "Invalid password",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 生成token
	token := GenerateToken()

	response := map[string]interface{}{
		"success":  true,
		"token":    token,
		"playerID": user.PlayerID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleSimpleRegister 简单注册处理
func (h *SimpleHTTPHandler) HandleSimpleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 检查用户是否已存在
	if h.userRepo.UserExists(req.Username) {
		response := map[string]interface{}{
			"success": false,
			"error":   "User already exists",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 创建新用户
	user := &common.UserData{
		Username: req.Username,
		Password: req.Password, // 注意：实际应用中应该存储密码哈希
		PlayerID: fmt.Sprintf("player_%d", len(req.Username)+1000),
	}

	if err := h.userRepo.SaveUser(user); err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to create user",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "User registered successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UserData 用户数据结构 - 这里使用common模块的类型
// type UserData struct {
// 	Username string
// 	Password string
// 	PlayerID string
// }
