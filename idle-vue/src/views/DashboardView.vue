<template>
  <div class="dashboard-view">
    <h1>用户控制台</h1>
    <p>欢迎，{{ userStore.username }}！</p>
    <p>游戏功能正在开发中...</p>
    <button @click="logout">退出登录</button>
  </div>
</template>

<script setup>
import { useUserStore } from '../store/user.js'

const userStore = useUserStore()

const logout = () => {
  userStore.logout()
}
</script>

<style scoped>
.dashboard-view {
  padding: 2rem;
  max-width: 800px;
  margin: 0 auto;
}

button {
  margin-top: 1rem;
  padding: 0.5rem 1rem;
  background-color: #dc3545;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

button:hover {
  background-color: #c82333;
}
</style>