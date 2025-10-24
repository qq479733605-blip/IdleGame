<script setup>
import axios from "axios";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useUserStore } from "../store/user";

const username = ref("");
const router = useRouter();
const user = useUserStore();
const isLoading = ref(false);

async function login() {
  if (!username.value.trim()) {
    alert("请输入道号！");
    return;
  }

  isLoading.value = true;
  try {
    const res = await axios.post("http://localhost:8080/login", {
      username: username.value,
      password: "default", // 发送默认密码
    });
    user.setUser(username.value, res.data.token);
    router.push("/main");
  } catch (e) {
    alert("登录失败：" + e.message);
  } finally {
    isLoading.value = false;
  }
}

function handleKeyPress(event) {
  if (event.key === 'Enter') {
    login();
  }
}
</script>

<template>
  <div class="login-container">
    <div class="login-background">
      <div class="stars"></div>
      <div class="mountains"></div>
    </div>

    <div class="login-panel">
      <div class="login-header">
        <h1 class="game-title">🏮 仙途凡尘</h1>
        <div class="game-subtitle">开启你的修仙之旅</div>
      </div>

      <div class="login-form">
        <div class="form-group">
          <label class="form-label">道号</label>
          <input
            v-model="username"
            @keypress="handleKeyPress"
            class="form-input"
            placeholder="请输入你的道号"
            :disabled="isLoading"
          />
        </div>

        <button
          @click="login"
          class="login-btn"
          :disabled="isLoading || !username.trim()"
        >
          <span v-if="isLoading" class="loading-text">
            <span class="loading-spinner">⚡</span>
            登录中...
          </span>
          <span v-else>
            <span class="btn-icon">🚀</span>
            开始修仙
          </span>
        </button>

        <div class="login-tips">
          <p>🌟 随心输入道号即可开始修仙</p>
          <p>📜 无需密码，一人一世界</p>
        </div>
      </div>

      <div class="login-footer">
        <div class="footer-text">
          <p>🔮 凡人亦可踏上仙途</p>
          <p>✨ 聚灵气，悟大道，成仙人</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  position: relative;
  height: 100vh;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Microsoft YaHei', sans-serif;
}

.login-background {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #0a0e27 0%, #1a1a2e 50%, #16213e 100%);
  z-index: -1;
}

.stars {
  position: absolute;
  width: 100%;
  height: 100%;
  background-image:
    radial-gradient(2px 2px at 20% 30%, white, transparent),
    radial-gradient(2px 2px at 60% 70%, white, transparent),
    radial-gradient(1px 1px at 50% 50%, white, transparent),
    radial-gradient(1px 1px at 80% 10%, white, transparent),
    radial-gradient(2px 2px at 90% 60%, white, transparent);
  background-size: 200% 200%;
  animation: stars 120s linear infinite;
  opacity: 0.3;
}

@keyframes stars {
  from { transform: translateY(0); }
  to { transform: translateY(-100%); }
}

.login-panel {
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(10px);
  border: 2px solid rgba(138, 43, 226, 0.3);
  border-radius: 20px;
  padding: 40px;
  max-width: 400px;
  width: 90%;
  box-shadow: 0 8px 32px rgba(138, 43, 226, 0.2);
  text-align: center;
  position: relative;
  z-index: 1;
}

.login-header {
  margin-bottom: 30px;
}

.game-title {
  font-size: 36px;
  background: linear-gradient(45deg, #ffd700, #ff6b6b, #4fc3f7);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin: 0 0 10px 0;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.game-subtitle {
  color: #b0b0b0;
  font-size: 16px;
  margin-bottom: 20px;
}

.login-form {
  margin-bottom: 30px;
}

.form-group {
  margin-bottom: 20px;
  text-align: left;
}

.form-label {
  display: block;
  color: #ffd700;
  font-size: 14px;
  margin-bottom: 8px;
  font-weight: bold;
}

.form-input {
  width: 100%;
  padding: 15px;
  background: rgba(255, 255, 255, 0.05);
  border: 2px solid rgba(255, 215, 0, 0.2);
  border-radius: 10px;
  color: #fff;
  font-size: 16px;
  transition: all 0.3s ease;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: #ffd700;
  background: rgba(255, 255, 255, 0.1);
  box-shadow: 0 0 20px rgba(255, 215, 0, 0.2);
}

.form-input::placeholder {
  color: #888;
}

.login-btn {
  width: 100%;
  padding: 15px;
  background: linear-gradient(45deg, #ffd700, #ff9800);
  border: none;
  border-radius: 10px;
  color: #000;
  font-size: 18px;
  font-weight: bold;
  cursor: pointer;
  transition: all 0.3s ease;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-top: 10px;
  position: relative;
  overflow: hidden;
}

.login-btn:hover:not(:disabled) {
  background: linear-gradient(45deg, #ff9800, #ff6b6b);
  transform: translateY(-2px);
  box-shadow: 0 5px 20px rgba(255, 215, 0, 0.4);
}

.login-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.loading-text {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.loading-spinner {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.btn-icon {
  margin-right: 8px;
}

.login-tips {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.login-tips p {
  color: #888;
  font-size: 14px;
  margin: 5px 0;
  font-style: italic;
}

.login-footer {
  margin-top: 20px;
}

.footer-text p {
  color: #666;
  font-size: 12px;
  margin: 3px 0;
  opacity: 0.7;
}

/* 响应式设计 */
@media (max-width: 480px) {
  .login-panel {
    padding: 30px 20px;
    margin: 20px;
  }

  .game-title {
    font-size: 28px;
  }

  .form-input {
    padding: 12px;
    font-size: 16px;
  }

  .login-btn {
    padding: 12px;
    font-size: 16px;
  }
}
</style>
