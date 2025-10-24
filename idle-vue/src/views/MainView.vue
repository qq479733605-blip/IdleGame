<script setup>
import { ref, onMounted, computed } from "vue";
import { useUserStore } from "../store/user";

const user = useUserStore();
const ws = ref(null);

const sequences = ref([]);
const selectedSeq = ref("");
const bag = ref({});
const gains = ref(0);
const isRunning = ref(false);
const logs = ref([]);
const playerLevel = ref(1);
const playerExp = ref(0);
const currentSeqLevel = ref(1);
const currentSeqExp = ref(0);
const seqProgress = ref(0);
const seqInterval = ref(3); // 默认3秒
const progressTimer = ref(null);
const itemNotifications = ref([]); // 物品获得通知
const notificationId = ref(0);
const showOfflineReward = ref(false); // 离线收益弹窗
const offlineRewardData = ref(null); // 离线收益数据
const showSeqReward = ref(false); // 序列结算弹窗
const seqRewardData = ref(null); // 序列结算数据
const seqLevels = ref({}); // 存储所有序列的等级信息
const isLoading = ref(true); // 加载状态
const showLoginScreen = ref(true); // 登录界面状态

onMounted(() => {
  // 显示加载动画
  setTimeout(() => {
    isLoading.value = false;
    connectWS();
  }, 2000);
});

function connectWS() {
  ws.value = new WebSocket(`ws://localhost:8080/ws?token=${user.token}`);

  ws.value.onopen = () => {
    logs.value.push("🌟 仙缘已定，开始你的修仙之旅！");
    ws.value.send(JSON.stringify({ type: "C_ListSeq" }));
  };

  ws.value.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    switch (msg.type) {
      case "S_LoginOK":
        playerExp.value = msg.exp || 0;
        logs.value.push(`🎊 ${user.username}道友，欢迎重返仙途！`);
        break;
      case "S_Reconnected":
        // 重连状态恢复
        console.log("收到 S_Reconnected 消息:", msg);
        logs.value.push(`🔄 ${msg.msg || "重连成功"}`);

        // 恢复玩家状态
        playerExp.value = msg.exp || 0;
        bag.value = msg.bag || {};

        // 保存重连状态，等待序列列表加载后再处理序列恢复
        if (msg.seq_id && msg.seq_level > 0) {
          // 将重连状态保存到一个临时变量
          window.pendingReconnectionState = {
            seq_id: msg.seq_id,
            seq_level: msg.seq_level,
            is_running: msg.is_running
          };

          selectedSeq.value = msg.seq_id;
          currentSeqLevel.value = msg.seq_level;

          // 更新序列等级信息
          if (msg.seq_levels) {
            seqLevels.value = msg.seq_levels;
          } else {
            seqLevels.value[msg.seq_id] = msg.seq_level;
          }

          // 如果序列正在运行，先设置运行状态，但延迟序列查找
          if (msg.is_running) {
            isRunning.value = true;
            // 尝试立即恢复进度条（如果序列已加载）
            tryRestoreSequenceProgress();
          } else {
            isRunning.value = false;
            stopProgressTimer(); // 确保停止进度条
          }
        } else {
          // 没有序列在运行，确保状态正确
          isRunning.value = false;
          stopProgressTimer();
        }
        break;
      case "S_OfflineReward":
        // 显示离线收益弹窗
        offlineRewardData.value = {
          gains: msg.gains || 0,
          duration: msg.offline_duration || 0,
          items: msg.offline_items || {}
        };
        showOfflineReward.value = true;
        break;
      case "S_ListSeq":
        sequences.value = msg.sequences;

        // 检查是否有待处理的重连状态需要恢复
        if (window.pendingReconnectionState) {
          handlePendingReconnection();
        } else if (sequences.value.length > 0) {
          // 只在没有重连状态时才默认选择第一个序列
          selectedSeq.value = sequences.value[0].id;
        }
        break;
      case "S_SeqResult":
        gains.value += msg.gains || 0;
        bag.value = msg.bag || {};

        if (msg.level && msg.seq_id === selectedSeq.value) {
          currentSeqLevel.value = msg.level;
          currentSeqExp.value = msg.cur_exp || 0;
        }

        // 更新序列等级信息
        if (msg.seq_id && msg.level) {
          seqLevels.value[msg.seq_id] = msg.level;
        }

        if (msg.rare && msg.rare.length > 0) {
          logs.value.push(`🌟 神秘书籍：${msg.rare.join(", ")}`);
        }

        if (msg.gains > 0) {
          logs.value.push(`💫 获得${msg.gains}点灵气`);
        }

        // 处理序列结算弹窗数据
        const newItems = {};
        if (msg.items && msg.items.length > 0) {
          msg.items.forEach(item => {
            const itemId = item.id;
            if (newItems[itemId]) {
              newItems[itemId].count++;
            } else {
              newItems[itemId] = {
                id: itemId,
                name: getItemName(itemId),
                icon: getItemIcon(itemId),
                count: 1
              };
            }
          });
        }

        // 显示序列结算弹窗（如果有收益或物品）
        if (msg.gains > 0 || Object.keys(newItems).length > 0) {
          seqRewardData.value = {
            gains: msg.gains || 0,
            items: Object.values(newItems),
            sequenceName: getSequenceName(msg.seq_id),
            rare: msg.rare || []
          };
          showSeqReward.value = true;

          // 2秒后自动隐藏弹窗
          setTimeout(() => {
            showSeqReward.value = false;
          }, 2000);
        }

        // 保持原有的物品通知功能
        Object.values(newItems).forEach(item => {
          addItemNotification(item);
        });
        break;
      case "S_SeqStarted":
        isRunning.value = true;
        currentSeqLevel.value = msg.level || 1;

        // 更新序列等级信息
        if (msg.seq_id && msg.level) {
          seqLevels.value[msg.seq_id] = msg.level;
        }

        logs.value.push(`🎯 开始${getSequenceName(msg.seq_id)} - 当前境界：${currentSeqLevel.value}重`);
        break;
      case "S_SeqEnded":
        isRunning.value = false;
        logs.value.push("⏸️ 暂停修炼，道法自然");
        break;
      default:
        console.log("Unhandled:", msg);
    }
  };

  ws.value.onclose = () => {
    logs.value.push("☁️ 仙缘暂断，重续仙缘中...");
    setTimeout(connectWS, 5000);
  };
}

function startSeq() {
  if (isRunning.value || !selectedSeq.value) return;

  // 获取选中序列的间隔时间
  const selectedSeqData = sequences.value.find(s => s.id === selectedSeq.value);
  if (selectedSeqData) {
    // 这里需要从后端获取实际的tick_interval，先用映射
    const intervalMap = {
      'meditation': 3,
      'herb_gathering': 4,
      'mining': 4,
      'alchemy': 5,
      'weapon_crafting': 6,
      'talisman_making': 4,
      'spirit_beast_taming': 5,
      'array_mastery': 6,
      'sword_practice': 4
    };
    seqInterval.value = intervalMap[selectedSeq.value] || 3;
    startProgressTimer();
  }

  ws.value.send(JSON.stringify({ type: "C_StartSeq", seq_id: selectedSeq.value, target: 100 }));
}

function startProgressTimer() {
  seqProgress.value = 0;
  clearInterval(progressTimer.value);

  progressTimer.value = setInterval(() => {
    seqProgress.value += (100 / (seqInterval.value * 10)); // 每100ms增加进度
    if (seqProgress.value >= 100) {
      seqProgress.value = 0; // 重置进度，等待后端结算
    }
  }, 100);
}

function stopProgressTimer() {
  clearInterval(progressTimer.value);
  seqProgress.value = 0;
}

// 尝试恢复序列进度（如果序列列表已加载）
function tryRestoreSequenceProgress() {
  if (!window.pendingReconnectionState) return;

  const pendingState = window.pendingReconnectionState;
  const seq = sequences.value.find(s => s.id === pendingState.seq_id);

  if (seq) {
    // 序列列表已加载，可以立即恢复
    restoreSequenceProgress(seq, pendingState);
    // 清除待处理状态
    window.pendingReconnectionState = null;
  }
  // 如果序列还没加载，等待 S_ListSeq 消息处理时再恢复
}

// 处理待处理的重连状态（在 S_ListSeq 后调用）
function handlePendingReconnection() {
  if (!window.pendingReconnectionState) return;

  const pendingState = window.pendingReconnectionState;
  const seq = sequences.value.find(s => s.id === pendingState.seq_id);

  if (seq) {
    // 确保选中正确的序列
    selectedSeq.value = pendingState.seq_id;

    if (pendingState.is_running) {
      restoreSequenceProgress(seq, pendingState);
    } else {
      // 如果序列没有在运行，确保停止进度条
      isRunning.value = false;
      stopProgressTimer();
    }
    // 清除待处理状态
    window.pendingReconnectionState = null;
  } else {
    // 如果找不到序列，重置状态
    console.warn(`重连时找不到序列: ${pendingState.seq_id}，重置为默认序列`);
    if (sequences.value.length > 0) {
      selectedSeq.value = sequences.value[0].id;
    }
    isRunning.value = false;
    stopProgressTimer();
    window.pendingReconnectionState = null;
  }
}

// 恢复序列进度的具体逻辑
function restoreSequenceProgress(seq, pendingState) {
  seqInterval.value = getSequenceInterval(seq.id); // 使用现有函数获取间隔
  seqProgress.value = Math.random() * 80; // 0-80%的随机进度
  startProgressTimer();
  logs.value.push(`♻️ 恢复修炼：${seq.name}，进度${Math.round(seqProgress.value)}%`);

  // 显示重连恢复提示
  seqRewardData.value = {
    gains: 0,
    items: [],
    sequenceName: `恢复${seq.name}修炼`,
    rare: []
  };
  showSeqReward.value = true;
  setTimeout(() => {
    showSeqReward.value = false;
  }, 1500);
}

function stopSeq() {
  stopProgressTimer();
  ws.value.send(JSON.stringify({ type: "C_StopSeq" }));
}

function addItemNotification(item) {
  const notification = {
    id: notificationId.value++,
    item: item,
    timestamp: Date.now()
  };
  itemNotifications.value.push(notification);

  // 1秒后自动移除通知
  setTimeout(() => {
    removeNotification(notification.id);
  }, 1000);
}

function removeNotification(id) {
  const index = itemNotifications.value.findIndex(n => n.id === id);
  if (index > -1) {
    itemNotifications.value.splice(index, 1);
  }
}

function confirmOfflineReward() {
  showOfflineReward.value = false;
  // 将离线收益应用到当前状态
  if (offlineRewardData.value) {
    gains.value += offlineRewardData.value.gains;
    // 合并离线物品到背包
    Object.entries(offlineRewardData.value.items).forEach(([itemId, count]) => {
      if (bag.value[itemId]) {
        bag.value[itemId] += count;
      } else {
        bag.value[itemId] = count;
      }
    });

    // 添加日志
    logs.value.push(`🌙 离线${offlineRewardData.value.duration}秒，获得${offlineRewardData.value.gains}点灵气`);

    // 显示物品通知
    Object.entries(offlineRewardData.value.items).forEach(([itemId, count]) => {
      for (let i = 0; i < count; i++) {
        addItemNotification({ id: itemId, name: getItemName(itemId), icon: getItemIcon(itemId), count: 1 });
      }
    });
  }
}

function getSequenceName(seqId) {
  const seq = sequences.value.find(s => s.id === seqId);
  return seq ? seq.name : seqId;
}

function getItemIcon(itemId) {
  const icons = {
    // 灵物类
    'herb_spirit': '🌿',
    'herb_rare': '🍄',
    'herb_legendary': '🌺',
    'flower_soul': '🌸',

    // 矿物类
    'ore_iron': '⛏️',
    'ore_copper': '🔶',
    'ore_silver': '🔷',
    'ore_gold': '🪙',
    'crystal_spirit': '💎',

    // 灵精类
    'essence_fire': '🔥',
    'essence_water': '💧',
    'essence_earth': '🪨',
    'essence_wind': '🌪️',

    // 丹药类
    'pill_low': '💊',
    'pill_mid': '🧪',
    'pill_high': '⚗️',
    'elixir_life': '🧬',

    // 武器法器类
    'sword_basic': '⚔️',
    'sword_spirit': '✨',
    'sword_divine': '🗡️',
    'armor_basic': '👘',
    'charm_protection': '🔮',

    // 符箓类
    'talisman_basic': '📜',
    'talisman_advanced': '🪄',
    'talisman_legendary': '📖',
    'scroll_ancient': '📜',

    // 灵兽类
    'beast_core': '🔴',
    'beast_soul': '👻',
    'beast_essence': '✨',
    'companion_egg': '🥚',

    // 阵法类
    'array_basic': '🔯',
    'array_advanced': '🎯',
    'array_legendary': '⭐',
    'rune_power': '🔠',

    // 剑道类
    'sword_intent': '💫',
    'sword_aura': '⚡',
    'sword_manual': '📚',
    'essence_sword': '🗡️'
  };
  return icons[itemId] || '📦';
}

function getItemName(itemId) {
  const names = {
    // 灵物类
    'herb_spirit': '灵草',
    'herb_rare': '千年灵芝',
    'herb_legendary': '仙界神草',
    'flower_soul': '魂花',

    // 矿物类
    'ore_iron': '玄铁矿',
    'ore_copper': '赤铜矿',
    'ore_silver': '皓银矿',
    'ore_gold': '金沙矿',
    'crystal_spirit': '灵晶石',

    // 灵精类
    'essence_fire': '火灵精',
    'essence_water': '水灵精',
    'essence_earth': '土灵精',
    'essence_wind': '风灵精',

    // 丹药类
    'pill_low': '筑基丹',
    'pill_mid': '金丹',
    'pill_high': '元婴丹',
    'elixir_life': '生命仙露',

    // 武器法器类
    'sword_basic': '基础法剑',
    'sword_spirit': '灵剑',
    'sword_divine': '仙剑',
    'armor_basic': '法袍',
    'charm_protection': '护身符',

    // 符箓类
    'talisman_basic': '基础符箓',
    'talisman_advanced': '高级符箓',
    'talisman_legendary': '传说符箓',
    'scroll_ancient': '古老卷轴',

    // 灵兽类
    'beast_core': '兽核',
    'beast_soul': '兽魂',
    'beast_essence': '灵兽精元',
    'companion_egg': '灵兽蛋',

    // 阵法类
    'array_basic': '基础阵盘',
    'array_advanced': '高级阵盘',
    'array_legendary': '传说阵图',
    'rune_power': '力量符文',

    // 剑道类
    'sword_intent': '剑意碎片',
    'sword_aura': '剑气',
    'sword_manual': '剑谱',
    'essence_sword': '剑灵精华'
  };
  return names[itemId] || itemId;
}

const cultivationRealm = computed(() => {
  const realms = [
    { level: 1, name: '凡人', desc: '芸芸众生，开始修仙之路' },
    { level: 5, name: '炼气', desc: '初窥门径，引气入体' },
    { level: 10, name: '筑基', desc: '筑下道基，真正的修仙者' },
    { level: 20, name: '金丹', desc: '凝结金丹，大道可期' },
    { level: 30, name: '元婴', desc: '元神出窍，逍遥天地' }
  ];

  for (let i = realms.length - 1; i >= 0; i--) {
    if (playerLevel.value >= realms[i].level) {
      return realms[i];
    }
  }
  return realms[0];
});

function getSequenceLevel(seqId) {
  return seqLevels.value[seqId] || 0;
}

function getSequenceInterval(seqId) {
  const intervalMap = {
    'meditation': 3,
    'herb_gathering': 4,
    'mining': 4,
    'alchemy': 5,
    'weapon_crafting': 6,
    'talisman_making': 4,
    'spirit_beast_taming': 5,
    'array_mastery': 6,
    'sword_practice': 4
  };
  return intervalMap[seqId] || 3;
}

function getInventoryItem(slot) {
  const items = Object.entries(bag.value);
  if (slot <= items.length && slot > 0) {
    const [itemId, count] = items[slot - 1];
    return { id: itemId, count: count };
  }
  return null;
}

function getSequenceIcon(seqId) {
  const icons = {
    'meditation': '🧘‍♂️',
    'herb_gathering': '🌿',
    'mining': '⛏️',
    'alchemy': '🧪',
    'weapon_crafting': '⚔️',
    'talisman_making': '📜',
    'spirit_beast_taming': '🐲',
    'array_mastery': '🔮',
    'sword_practice': '⚡'
  };
  return icons[seqId] || '🎯';
}

function getSequenceDesc(seqId) {
  const descs = {
    'meditation': '静心凝神，领悟天地大道',
    'herb_gathering': '深山采药，收集天地灵草',
    'mining': '开山采石，获取灵矿宝玉',
    'alchemy': '炼制丹药，提升修为境界',
    'weapon_crafting': '锻造法器，增强战力',
    'talisman_making': '绘制符箓，获得神秘力量',
    'spirit_beast_taming': '驯养灵兽，得道相助',
    'array_mastery': '研究阵法，掌握天地之力',
    'sword_practice': '剑道修行，磨练战斗技巧'
  };
  return descs[seqId] || '神秘的修炼法门';
}
</script>

<template>
  <!-- 加载界面 -->
  <div v-if="isLoading" class="loading-screen">
    <div class="loading-content">
      <div class="loading-logo">
        <div class="logo-icon">🧘‍♂️</div>
        <div class="logo-text">修仙放置</div>
      </div>
      <div class="loading-animation">
        <div class="loading-bar"></div>
        <div class="loading-text">正在连接仙界...</div>
      </div>
      <div class="loading-particles">
        <div class="particle" v-for="i in 12" :key="i" :style="`--i: ${i}`"></div>
      </div>
    </div>
  </div>

  <!-- 游戏主界面 -->
  <div v-else class="cultivation-game">
    <!-- 顶部境界信息 -->
    <div class="realm-header">
      <div class="player-info">
        <h1 class="player-title">🧘‍♂️ {{ user.username }}道友</h1>
        <div class="realm-info">
          <div class="realm-name">{{ cultivationRealm.name }}</div>
          <div class="realm-desc">{{ cultivationRealm.desc }}</div>
        </div>
      </div>
      <div class="stats-panel">
        <div class="stat-item">
          <span class="stat-label">灵气</span>
          <span class="stat-value">{{ gains }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">修为</span>
          <span class="stat-value">{{ playerExp }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">当前序列</span>
          <span class="stat-value">{{ currentSeqLevel }}重</span>
        </div>
      </div>
    </div>

    <!-- 修炼进度条 -->
    <div v-if="isRunning" class="progress-panel">
      <h3 class="progress-title">
        ⚡ {{ getSequenceName(selectedSeq) }}
        <span v-if="currentSeqLevel > 0" class="progress-level">{{ currentSeqLevel }}重</span>
        <span class="progress-divider">|</span>
        <span class="progress-label">进度</span>
      </h3>
      <div class="progress-bar-container">
        <div class="progress-bar" :style="{ width: seqProgress + '%' }"></div>
        <div class="progress-text">{{ Math.round(seqProgress) }}%</div>
      </div>
    </div>

    <!-- 物品获得通知 -->
    <div class="notifications-container">
      <div
        v-for="notification in itemNotifications"
        :key="notification.id"
        class="item-notification"
      >
        <div class="notification-icon">{{ notification.item.icon }}</div>
        <div class="notification-content">
          <div class="notification-title">获得新物品！</div>
          <div class="notification-name">{{ notification.item.name }} ×{{ notification.item.count }}</div>
        </div>
      </div>
    </div>

    <!-- 修炼选择区域 -->
    <div class="cultivation-panel">
      <h2 class="panel-title">🔮 修炼法门</h2>
      <div class="sequence-grid">
        <div
          v-for="s in sequences"
          :key="s.id"
          class="sequence-card"
          :class="{ active: selectedSeq === s.id, running: isRunning && selectedSeq === s.id }"
          @click="!isRunning && (selectedSeq = s.id)"
        >
          <div class="sequence-icon">
            {{ getSequenceIcon(s.id) }}
            <div v-if="getSequenceLevel(s.id) > 0" class="sequence-level-badge">
              {{ getSequenceLevel(s.id) }}重
            </div>
          </div>
          <div class="sequence-name">{{ s.name }}</div>
          <div class="sequence-desc">{{ getSequenceDesc(s.id) }}</div>
          <div class="sequence-time">{{ getSequenceInterval(s.id) }}秒/次</div>
        </div>
      </div>

      <div class="action-buttons">
        <button
          v-if="!isRunning"
          @click="startSeq"
          class="btn btn-primary"
          :disabled="!selectedSeq"
        >
          🚀 开始修炼
        </button>
        <button
          v-else
          @click="stopSeq"
          class="btn btn-danger"
        >
          ⏸️ 停止修炼
        </button>
      </div>
    </div>

    <!-- 格子背包界面 -->
    <div class="inventory-panel">
      <h2 class="panel-title">🎒 乾坤袋</h2>
      <div class="inventory-slots">
        <div
          v-for="slot in 24"
          :key="'slot-' + slot"
          class="inventory-slot"
        >
          <div v-if="getInventoryItem(slot)" class="slot-item">
            <div class="slot-icon">{{ getItemIcon(getInventoryItem(slot).id) }}</div>
            <div class="slot-count">{{ getInventoryItem(slot).count }}</div>
            <div class="slot-name">{{ getItemName(getInventoryItem(slot).id) }}</div>
          </div>
          <div v-else class="empty-slot">空</div>
        </div>
      </div>
    </div>

    <!-- 修炼日志 -->
    <div class="log-panel">
      <h2 class="panel-title">📜 修炼日志</h2>
      <div class="log-container">
        <div
          v-for="(log, index) in logs.slice(-15).reverse()"
          :key="index"
          class="log-entry"
        >
          {{ log }}
        </div>
      </div>
    </div>

    <!-- 序列结算透明弹窗 -->
    <div v-if="showSeqReward && seqRewardData && (seqRewardData.gains > 0 || seqRewardData.items.length > 0 || seqRewardData.rare.length > 0 || seqRewardData.sequenceName?.includes('恢复'))" class="seq-reward-popup">
      <div class="seq-reward-content" :class="{ 'recovery-popup': seqRewardData.sequenceName?.includes('恢复') }">
        <div class="seq-reward-header">
          <div class="seq-reward-title">
            {{ seqRewardData?.sequenceName?.includes('恢复') ? '🔄 ' : '✨ ' }}{{ seqRewardData?.sequenceName }}{{ seqRewardData?.sequenceName?.includes('恢复') ? '状态恢复' : '修仙收获' }}
          </div>
        </div>

        <div v-if="seqRewardData" class="seq-reward-items">
          <!-- 灵气收益 -->
          <div v-if="seqRewardData.gains > 0" class="seq-reward-gains">
            <div class="gains-icon">💫</div>
            <div class="gains-text">+{{ seqRewardData.gains }} 灵气</div>
          </div>

          <!-- 物品收益 -->
          <div v-if="seqRewardData.items.length > 0" class="seq-items-list">
            <div
              v-for="item in seqRewardData.items"
              :key="item.id"
              class="seq-item-display"
            >
              <div class="seq-item-icon">{{ item.icon }}</div>
              <div class="seq-item-info">
                <div class="seq-item-name">{{ item.name }}</div>
                <div class="seq-item-count">×{{ item.count }}</div>
              </div>
            </div>
          </div>

          <!-- 神秘事件 -->
          <div v-if="seqRewardData.rare.length > 0" class="seq-rare-event">
            <div class="rare-icon">🌟</div>
            <div class="rare-text">{{ seqRewardData.rare.join(", ") }}</div>
          </div>

          <!-- 恢复提示 -->
          <div v-if="seqRewardData.sequenceName?.includes('恢复')" class="seq-recovery-message">
            <div class="recovery-icon">♻️</div>
            <div class="recovery-text">修炼进度已恢复</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 离线收益弹窗 -->
    <div v-if="showOfflineReward" class="offline-reward-overlay" @click.self="confirmOfflineReward">
      <div class="offline-reward-modal">
        <div class="reward-header">
          <h3 class="reward-title">🌙 离线收益</h3>
          <div class="reward-subtitle">你离线期间的收获</div>
        </div>

        <div v-if="offlineRewardData" class="reward-content">
          <div class="reward-stats">
            <div class="stat-item">
              <span class="stat-label">离线时长</span>
              <span class="stat-value">{{ Math.floor(offlineRewardData.duration / 60) }}分钟</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">灵气收益</span>
              <span class="stat-value">{{ offlineRewardData.gains }}</span>
            </div>
          </div>

          <div v-if="Object.keys(offlineRewardData.items).length > 0" class="reward-items">
            <div class="items-title">获得物品：</div>
            <div class="items-grid">
              <div
                v-for="(count, item) in offlineRewardData.items"
                :key="item"
                class="reward-item"
              >
                <div class="reward-icon">{{ getItemIcon(item) }}</div>
                <div class="reward-name">{{ getItemName(item) }}</div>
                <div class="reward-count">×{{ count }}</div>
              </div>
            </div>
          </div>

          <div v-else class="no-items">
            本次离线未获得特殊物品
          </div>
        </div>

        <div class="reward-actions">
          <button @click="confirmOfflineReward" class="reward-confirm-btn">
            🎯 确认收益
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cultivation-game {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
  min-height: 100vh;
  color: #e8e8e8;
  font-family: 'Microsoft YaHei', sans-serif;
}

/* 顶部境界信息 */
.realm-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 15px;
  padding: 20px;
  margin-bottom: 25px;
  border: 2px solid rgba(138, 43, 226, 0.3);
  box-shadow: 0 8px 32px rgba(138, 43, 226, 0.2);
}

.player-title {
  font-size: 28px;
  margin: 0;
  background: linear-gradient(45deg, #ffd700, #ff6b6b);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.realm-name {
  font-size: 20px;
  font-weight: bold;
  color: #ffd700;
  margin-top: 5px;
}

.realm-desc {
  font-size: 14px;
  color: #b0b0b0;
  margin-top: 5px;
}

.stats-panel {
  display: flex;
  gap: 20px;
}

.stat-item {
  text-align: center;
  background: rgba(255, 255, 255, 0.05);
  padding: 15px;
  border-radius: 10px;
  border: 1px solid rgba(138, 43, 226, 0.2);
}

.stat-label {
  display: block;
  font-size: 12px;
  color: #b0b0b0;
  margin-bottom: 5px;
}

.stat-value {
  display: block;
  font-size: 18px;
  font-weight: bold;
  color: #4fc3f7;
}

/* 面板通用样式 */
.panel-title {
  font-size: 20px;
  margin: 0 0 20px 0;
  color: #ffd700;
  text-align: center;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.5);
}

/* 修炼面板 */
.cultivation-panel {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 15px;
  padding: 25px;
  margin-bottom: 25px;
  border: 2px solid rgba(76, 175, 80, 0.3);
}

.sequence-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 15px;
  margin-bottom: 25px;
}

.sequence-card {
  background: rgba(255, 255, 255, 0.08);
  border: 2px solid rgba(76, 175, 80, 0.2);
  border-radius: 12px;
  padding: 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.sequence-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.1), transparent);
  transition: left 0.6s ease;
}

.sequence-card:hover::before {
  left: 100%;
}

.sequence-card:hover {
  background: rgba(255, 255, 255, 0.15);
  border-color: rgba(76, 175, 80, 0.6);
  transform: translateY(-4px) scale(1.02);
  box-shadow: 0 8px 25px rgba(76, 175, 80, 0.4),
              0 0 20px rgba(76, 175, 80, 0.2),
              inset 0 0 15px rgba(76, 175, 80, 0.1);
}

.sequence-card:active {
  transform: translateY(-2px) scale(0.98);
  transition: all 0.1s ease;
}

.sequence-card.active {
  background: rgba(76, 175, 80, 0.2);
  border-color: #4caf50;
  box-shadow: 0 0 20px rgba(76, 175, 80, 0.4);
}

.sequence-card.running {
  animation: pulse 2s infinite;
  background: rgba(255, 193, 7, 0.2);
  border-color: #ffc107;
}

@keyframes pulse {
  0% { box-shadow: 0 0 20px rgba(255, 193, 7, 0.4); }
  50% { box-shadow: 0 0 30px rgba(255, 193, 7, 0.8); }
  100% { box-shadow: 0 0 20px rgba(255, 193, 7, 0.4); }
}

.sequence-icon {
  font-size: 40px;
  margin-bottom: 10px;
  position: relative;
  display: inline-block;
  transition: all 0.3s ease;
  animation: iconFloat 3s ease-in-out infinite;
}

@keyframes iconFloat {
  0%, 100% { transform: translateY(0px); }
  50% { transform: translateY(-3px); }
}

.sequence-card:hover .sequence-icon {
  transform: scale(1.1) rotate(5deg);
  animation: none;
  filter: drop-shadow(0 0 10px rgba(76, 175, 80, 0.6));
}

.sequence-level-badge {
  position: absolute;
  top: -8px;
  right: -12px;
  background: linear-gradient(45deg, #ff6b6b, #ff8e53);
  color: white;
  font-size: 10px;
  font-weight: bold;
  padding: 2px 6px;
  border-radius: 10px;
  box-shadow: 0 2px 6px rgba(255, 107, 107, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.3);
  white-space: nowrap;
  z-index: 10;
  animation: badgePulse 2s ease-in-out infinite;
}

@keyframes badgePulse {
  0%, 100% {
    transform: scale(1);
    box-shadow: 0 2px 6px rgba(255, 107, 107, 0.4);
  }
  50% {
    transform: scale(1.05);
    box-shadow: 0 2px 8px rgba(255, 107, 107, 0.6);
  }
}

.sequence-name {
  font-size: 16px;
  font-weight: bold;
  color: #fff;
  margin-bottom: 8px;
}

.sequence-desc {
  font-size: 12px;
  color: #b0b0b0;
  line-height: 1.4;
}

.action-buttons {
  text-align: center;
}

.btn {
  padding: 12px 30px;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: bold;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  text-transform: uppercase;
  letter-spacing: 1px;
  position: relative;
  overflow: hidden;
}

.btn::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 0;
  height: 0;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.3);
  transform: translate(-50%, -50%);
  transition: width 0.6s ease, height 0.6s ease;
}

.btn:active::before {
  width: 300px;
  height: 300px;
}

.btn-primary {
  background: linear-gradient(45deg, #4caf50, #45a049);
  color: white;
  box-shadow: 0 4px 15px rgba(76, 175, 80, 0.3);
  animation: btnPulse 3s ease-in-out infinite;
}

@keyframes btnPulse {
  0%, 100% {
    box-shadow: 0 4px 15px rgba(76, 175, 80, 0.3);
  }
  50% {
    box-shadow: 0 4px 20px rgba(76, 175, 80, 0.5);
  }
}

.btn-primary:hover:not(:disabled) {
  background: linear-gradient(45deg, #45a049, #3d8b40);
  transform: translateY(-3px) scale(1.05);
  box-shadow: 0 8px 25px rgba(76, 175, 80, 0.5),
              0 0 20px rgba(76, 175, 80, 0.3);
}

.btn-primary:active:not(:disabled) {
  transform: translateY(-1px) scale(0.98);
  transition: all 0.1s ease;
}

.btn-danger {
  background: linear-gradient(45deg, #f44336, #d32f2f);
  color: white;
  box-shadow: 0 4px 15px rgba(244, 67, 54, 0.3);
}

.btn-danger:hover:not(:disabled) {
  background: linear-gradient(45deg, #d32f2f, #c62828);
  transform: translateY(-3px) scale(1.05);
  box-shadow: 0 8px 25px rgba(244, 67, 54, 0.5),
              0 0 20px rgba(244, 67, 54, 0.3);
}

.btn-danger:active:not(:disabled) {
  transform: translateY(-1px) scale(0.98);
  transition: all 0.1s ease;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

/* 背包面板 */
.inventory-panel {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 15px;
  padding: 25px;
  margin-bottom: 25px;
  border: 2px solid rgba(255, 152, 0, 0.3);
}

.inventory-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 15px;
}

.inventory-item {
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 152, 0, 0.2);
  border-radius: 10px;
  padding: 15px 10px;
  text-align: center;
  transition: all 0.3s ease;
}

.inventory-item:hover {
  background: rgba(255, 255, 255, 0.12);
  transform: scale(1.05);
}

.item-icon {
  font-size: 30px;
  margin-bottom: 8px;
}

.item-name {
  font-size: 12px;
  color: #fff;
  margin-bottom: 5px;
  font-weight: bold;
}

.item-count {
  font-size: 14px;
  color: #ff9800;
  font-weight: bold;
}

.empty-inventory {
  text-align: center;
  color: #b0b0b0;
  font-style: italic;
  padding: 40px;
}

/* 日志面板 */
.log-panel {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 15px;
  padding: 25px;
  border: 2px solid rgba(158, 158, 158, 0.3);
}

.log-container {
  height: 200px;
  overflow-y: auto;
  background: rgba(0, 0, 0, 0.3);
  border-radius: 10px;
  padding: 15px;
}

.log-entry {
  font-size: 14px;
  color: #e0e0e0;
  margin-bottom: 8px;
  padding: 5px 10px;
  background: rgba(255, 255, 255, 0.05);
  border-left: 3px solid #4caf50;
  border-radius: 3px;
}

.log-entry:nth-child(even) {
  background: rgba(255, 255, 255, 0.08);
  border-left-color: #ff9800;
}

/* 滚动条样式 */
::-webkit-scrollbar {
  width: 8px;
}

::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.3);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.5);
}

/* 进度条样式 */
.progress-panel {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 15px;
  padding: 20px;
  margin-bottom: 25px;
  border: 2px solid rgba(255, 193, 7, 0.3);
  text-align: center;
  position: relative;
  overflow: hidden;
  animation: panelPulse 4s ease-in-out infinite;
}

.progress-panel::before {
  content: '';
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(255, 193, 7, 0.1) 0%, transparent 70%);
  animation: backgroundRotate 20s linear infinite;
}

@keyframes panelPulse {
  0%, 100% {
    box-shadow: 0 0 20px rgba(255, 193, 7, 0.3),
                inset 0 0 15px rgba(255, 193, 7, 0.1);
  }
  50% {
    box-shadow: 0 0 30px rgba(255, 193, 7, 0.5),
                inset 0 0 20px rgba(255, 193, 7, 0.15);
  }
}

@keyframes backgroundRotate {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.progress-title {
  color: #ffc107;
  margin: 0 0 15px 0;
  font-size: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.progress-level {
  background: linear-gradient(45deg, #ff6b6b, #ff8e53);
  color: white;
  font-size: 12px;
  font-weight: bold;
  padding: 2px 8px;
  border-radius: 12px;
  box-shadow: 0 2px 4px rgba(255, 107, 107, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.progress-divider {
  color: #888;
  font-size: 14px;
  margin: 0 4px;
}

.progress-label {
  color: #fff;
  font-weight: normal;
}

.progress-bar-container {
  position: relative;
  width: 100%;
  height: 25px;
  background: rgba(0, 0, 0, 0.3);
  border-radius: 15px;
  overflow: hidden;
  border: 1px solid rgba(255, 193, 7, 0.2);
}

.progress-bar {
  height: 100%;
  background: linear-gradient(90deg, #ffc107, #ff9800);
  border-radius: 15px;
  transition: width 0.1s linear;
  box-shadow: 0 0 10px rgba(255, 193, 7, 0.5);
  position: relative;
  overflow: hidden;
  animation: progressGlow 2s ease-in-out infinite;
}

.progress-bar::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.4), transparent);
  animation: progressShine 3s linear infinite;
}

@keyframes progressGlow {
  0%, 100% {
    box-shadow: 0 0 10px rgba(255, 193, 7, 0.5),
                0 0 20px rgba(255, 193, 7, 0.3),
                inset 0 0 10px rgba(255, 255, 255, 0.2);
  }
  50% {
    box-shadow: 0 0 20px rgba(255, 193, 7, 0.8),
                0 0 30px rgba(255, 193, 7, 0.5),
                inset 0 0 15px rgba(255, 255, 255, 0.3);
  }
}

@keyframes progressShine {
  0% {
    left: -100%;
  }
  100% {
    left: 100%;
  }
}

.progress-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: white;
  font-weight: bold;
  font-size: 12px;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

/* 物品通知样式 */
.notifications-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 1000;
  pointer-events: none; /* 不阻止鼠标事件 */
}

.item-notification {
  background: rgba(0, 0, 0, 0.7);
  border-left: 4px solid #4caf50;
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 12px;
  animation: notificationSlide 1.2s cubic-bezier(0.4, 0, 0.2, 1) forwards;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(5px);
  min-width: 200px;
  position: relative;
  overflow: hidden;
}

.item-notification::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(76, 175, 80, 0.2), transparent);
  animation: notificationShine 1.5s ease-out;
}

@keyframes notificationSlide {
  0% {
    transform: translateX(100%) translateY(20px) scale(0.8);
    opacity: 0;
  }
  20% {
    transform: translateX(-10px) translateY(0) scale(1.05);
    opacity: 1;
  }
  30% {
    transform: translateX(5px) translateY(0) scale(1);
  }
  40% {
    transform: translateX(0) translateY(0) scale(1);
  }
  80% {
    transform: translateX(0) translateY(0) scale(1);
    opacity: 1;
  }
  100% {
    transform: translateX(0) translateY(-10px) scale(0.95);
    opacity: 0;
  }
}

@keyframes notificationShine {
  0% {
    left: -100%;
  }
  100% {
    left: 100%;
  }
}

.notification-icon {
  font-size: 30px;
  flex-shrink: 0;
  animation: iconBounce 1.2s cubic-bezier(0.68, -0.55, 0.265, 1.55);
}

@keyframes iconBounce {
  0% {
    transform: scale(0.3) rotate(-15deg);
  }
  30% {
    transform: scale(1.2) rotate(5deg);
  }
  50% {
    transform: scale(0.9) rotate(-2deg);
  }
  70% {
    transform: scale(1.05) rotate(1deg);
  }
  100% {
    transform: scale(1) rotate(0deg);
  }
}

.notification-content {
  flex: 1;
}

.notification-title {
  font-size: 14px;
  font-weight: bold;
  color: #fff;
  margin-bottom: 3px;
}

.notification-name {
  font-size: 13px;
  color: #e8f5e8;
}

/* 序列时间显示 */
.sequence-time {
  font-size: 11px;
  color: #ff9800;
  background: rgba(255, 152, 0, 0.1);
  padding: 3px 8px;
  border-radius: 10px;
  margin-top: 8px;
  font-weight: bold;
}

/* 格子背包样式 */
.inventory-slots {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 8px;
  padding: 15px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 10px;
}

.inventory-slot {
  aspect-ratio: 1;
  background: rgba(255, 255, 255, 0.05);
  border: 2px solid rgba(255, 152, 0, 0.2);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  position: relative;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.inventory-slot::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, transparent, rgba(255, 152, 0, 0.1), transparent);
  transform: translateX(-100%);
  transition: transform 0.6s ease;
}

.inventory-slot:hover::before {
  transform: translateX(100%);
}

.inventory-slot:hover {
  background: rgba(255, 255, 255, 0.15);
  border-color: rgba(255, 152, 0, 0.6);
  transform: scale(1.08) translateY(-2px);
  box-shadow: 0 6px 20px rgba(255, 152, 0, 0.4),
              0 0 15px rgba(255, 152, 0, 0.2),
              inset 0 0 10px rgba(255, 152, 0, 0.1);
}

.inventory-slot:active {
  transform: scale(1.02) translateY(-1px);
  transition: all 0.1s ease;
}

.slot-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  width: 100%;
  text-align: center;
}

.slot-icon {
  font-size: 24px;
  margin-bottom: 3px;
  transition: all 0.3s ease;
  animation: slotIconFloat 4s ease-in-out infinite;
}

@keyframes slotIconFloat {
  0%, 100% { transform: translateY(0px) rotate(0deg); }
  25% { transform: translateY(-2px) rotate(2deg); }
  50% { transform: translateY(0px) rotate(0deg); }
  75% { transform: translateY(-1px) rotate(-1deg); }
}

.inventory-slot:hover .slot-icon {
  transform: scale(1.2) rotate(10deg);
  animation: none;
  filter: drop-shadow(0 0 8px rgba(255, 152, 0, 0.6));
}

.slot-count {
  font-size: 10px;
  color: #ff9800;
  font-weight: bold;
  position: absolute;
  top: 2px;
  right: 2px;
  background: rgba(0, 0, 0, 0.7);
  padding: 2px 4px;
  border-radius: 3px;
}

.slot-name {
  font-size: 9px;
  color: #e0e0e0;
  margin-top: 2px;
  font-weight: bold;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-slot {
  color: #666;
  font-size: 11px;
  font-style: italic;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .realm-header {
    flex-direction: column;
    text-align: center;
    gap: 20px;
  }

  .stats-panel {
    flex-wrap: wrap;
    justify-content: center;
  }

  .sequence-grid {
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  }

  .inventory-slots {
    grid-template-columns: repeat(4, 1fr);
  }

  .notifications-container {
    right: 10px;
    top: 10px;
    max-width: 250px;
  }
}

/* 离线收益弹窗样式 */
.offline-reward-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(5px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.offline-reward-modal {
  background: linear-gradient(135deg, rgba(26, 26, 46, 0.95), rgba(22, 33, 62, 0.95));
  border: 3px solid #ffd700;
  border-radius: 20px;
  padding: 30px;
  max-width: 450px;
  width: 90%;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  animation: modalSlideIn 0.3s ease-out;
  text-align: center;
}

@keyframes modalSlideIn {
  from {
    transform: scale(0.8);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
  }
}

.reward-header {
  margin-bottom: 20px;
  text-align: center;
}

.reward-title {
  font-size: 24px;
  background: linear-gradient(45deg, #ffd700, #ff9800);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin: 0 0 8px 0;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.reward-subtitle {
  color: #b0b0b0;
  font-size: 16px;
  font-style: italic;
}

.reward-content {
  margin-bottom: 25px;
  text-align: left;
}

.reward-stats {
  display: flex;
  gap: 30px;
  justify-content: center;
  margin-bottom: 20px;
}

.reward-stat-item {
  text-align: center;
}

.stat-label {
  display: block;
  color: #888;
  font-size: 14px;
  margin-bottom: 5px;
}

.stat-value {
  display: block;
  color: #4fc3f7;
  font-size: 20px;
  font-weight: bold;
}

.reward-items {
  margin-bottom: 20px;
}

.items-title {
  color: #ffd700;
  font-size: 16px;
  font-weight: bold;
  margin-bottom: 15px;
  text-align: center;
}

.items-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(80px, 1fr));
  gap: 12px;
}

.reward-item {
  background: rgba(255, 255, 255, 0.05);
  border: 2px solid rgba(255, 215, 0, 0.3);
  border-radius: 10px;
  padding: 12px;
  text-align: center;
  transition: all 0.3s ease;
}

.reward-item:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #ffd700;
  transform: scale(1.05);
}

.reward-icon {
  font-size: 24px;
  margin-bottom: 5px;
}

.reward-name {
  font-size: 12px;
  color: #fff;
  margin-bottom: 3px;
  font-weight: bold;
}

.reward-count {
  font-size: 14px;
  color: #ff9800;
  font-weight: bold;
}

.no-items {
  color: #888;
  font-style: italic;
  text-align: center;
  padding: 20px;
  font-size: 14px;
}

.reward-actions {
  text-align: center;
}

.reward-confirm-btn {
  background: linear-gradient(45deg, #ffd700, #ff9800);
  border: none;
  border-radius: 10px;
  padding: 15px 30px;
  font-size: 16px;
  font-weight: bold;
  color: #000;
  cursor: pointer;
  transition: all 0.3s ease;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.reward-confirm-btn:hover {
  background: linear-gradient(45deg, #ff9800, #ff6b6b);
  transform: translateY(-2px);
  box-shadow: 0 5px 20px rgba(255, 215, 0, 0.4);
}

/* 序列结算透明弹窗样式 */
.seq-reward-popup {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 1500;
  pointer-events: none; /* 不阻止鼠标事件 */
  animation: seqRewardFadeIn 0.3s ease-out;
}

@keyframes seqRewardFadeIn {
  0% {
    opacity: 0;
    transform: translate(-50%, -50%) scale(0.8);
  }
  50% {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1.05);
  }
  100% {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
}

.seq-reward-content {
  background: linear-gradient(135deg, rgba(0, 0, 0, 0.85), rgba(20, 20, 40, 0.85));
  backdrop-filter: blur(10px);
  border: 2px solid rgba(255, 215, 0, 0.6);
  border-radius: 15px;
  padding: 20px 25px;
  min-width: 280px;
  max-width: 400px;
  box-shadow: 0 8px 32px rgba(255, 215, 0, 0.3),
              0 0 20px rgba(255, 215, 0, 0.1);
}

.seq-reward-content.recovery-popup {
  border-color: rgba(76, 175, 80, 0.6);
  box-shadow: 0 8px 32px rgba(76, 175, 80, 0.3),
              0 0 20px rgba(76, 175, 80, 0.1);
}

.seq-reward-header {
  text-align: center;
  margin-bottom: 15px;
  border-bottom: 1px solid rgba(255, 215, 0, 0.3);
  padding-bottom: 10px;
}

.seq-reward-title {
  font-size: 18px;
  font-weight: bold;
  background: linear-gradient(45deg, #ffd700, #ff9800);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.seq-reward-items {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.seq-reward-gains {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(76, 175, 80, 0.2);
  border-radius: 10px;
  padding: 10px 15px;
  border: 1px solid rgba(76, 175, 80, 0.4);
}

.gains-icon {
  font-size: 24px;
  flex-shrink: 0;
}

.gains-text {
  font-size: 16px;
  font-weight: bold;
  color: #4caf50;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

.seq-items-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 200px;
  overflow-y: auto;
}

.seq-item-display {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  padding: 8px 12px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  transition: all 0.2s ease;
}

.seq-item-display:hover {
  background: rgba(255, 255, 255, 0.15);
  border-color: rgba(255, 215, 0, 0.4);
}

.seq-item-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.seq-item-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.seq-item-name {
  font-size: 14px;
  color: #fff;
  font-weight: bold;
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.seq-item-count {
  font-size: 13px;
  color: #ff9800;
  font-weight: bold;
  flex-shrink: 0;
  margin-left: 8px;
}

.seq-rare-event {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(156, 39, 176, 0.2);
  border-radius: 10px;
  padding: 10px 15px;
  border: 1px solid rgba(156, 39, 176, 0.4);
}

.rare-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.rare-text {
  font-size: 14px;
  color: #e1bee7;
  font-weight: bold;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

.seq-recovery-message {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(76, 175, 80, 0.2);
  border-radius: 10px;
  padding: 10px 15px;
  border: 1px solid rgba(76, 175, 80, 0.4);
}

.recovery-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.recovery-text {
  font-size: 14px;
  color: #a5d6a7;
  font-weight: bold;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

/* 序列弹窗滚动条 */
.seq-items-list::-webkit-scrollbar {
  width: 4px;
}

.seq-items-list::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 2px;
}

.seq-items-list::-webkit-scrollbar-thumb {
  background: rgba(255, 215, 0, 0.4);
  border-radius: 2px;
}

.seq-items-list::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 215, 0, 0.6);
}

/* 响应式序列弹窗 */
@media (max-width: 480px) {
  .seq-reward-content {
    min-width: 260px;
    max-width: 90vw;
    padding: 15px 20px;
  }

  .seq-reward-title {
    font-size: 16px;
  }

  .seq-item-display {
    padding: 6px 10px;
  }

  .seq-item-name {
    font-size: 13px;
  }

  .seq-item-count {
    font-size: 12px;
  }
}

/* 加载界面样式 */
.loading-screen {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  overflow: hidden;
}

.loading-content {
  text-align: center;
  position: relative;
  z-index: 10;
}

.loading-logo {
  margin-bottom: 50px;
  animation: logoFloat 3s ease-in-out infinite;
}

@keyframes logoFloat {
  0%, 100% { transform: translateY(0px); }
  50% { transform: translateY(-10px); }
}

.logo-icon {
  font-size: 80px;
  margin-bottom: 20px;
  animation: iconGlow 2s ease-in-out infinite alternate;
}

@keyframes iconGlow {
  0% { filter: drop-shadow(0 0 20px rgba(255, 215, 0, 0.6)); }
  100% { filter: drop-shadow(0 0 40px rgba(255, 215, 0, 0.9)); }
}

.logo-text {
  font-size: 36px;
  font-weight: bold;
  background: linear-gradient(45deg, #ffd700, #ff6b6b, #4fc3f7);
  background-size: 200% 200%;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  animation: textGradient 3s ease-in-out infinite;
}

@keyframes textGradient {
  0%, 100% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
}

.loading-animation {
  margin-bottom: 30px;
}

.loading-bar {
  width: 200px;
  height: 4px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 2px;
  margin: 0 auto 20px;
  overflow: hidden;
  position: relative;
}

.loading-bar::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, #ffd700, transparent);
  animation: loadingBar 2s ease-in-out infinite;
}

@keyframes loadingBar {
  0% { left: -100%; }
  100% { left: 100%; }
}

.loading-text {
  color: #b0b0b0;
  font-size: 16px;
  animation: textPulse 2s ease-in-out infinite;
}

@keyframes textPulse {
  0%, 100% { opacity: 0.6; }
  50% { opacity: 1; }
}

.loading-particles {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 300px;
  height: 300px;
  transform: translate(-50%, -50%);
  pointer-events: none;
}

.particle {
  position: absolute;
  width: 4px;
  height: 4px;
  background: #ffd700;
  border-radius: 50%;
  opacity: 0;
  animation: particleAnimation 4s ease-in-out infinite;
}

.particle:nth-child(1) { --i: 1; }
.particle:nth-child(2) { --i: 2; }
.particle:nth-child(3) { --i: 3; }
.particle:nth-child(4) { --i: 4; }
.particle:nth-child(5) { --i: 5; }
.particle:nth-child(6) { --i: 6; }
.particle:nth-child(7) { --i: 7; }
.particle:nth-child(8) { --i: 8; }
.particle:nth-child(9) { --i: 9; }
.particle:nth-child(10) { --i: 10; }
.particle:nth-child(11) { --i: 11; }
.particle:nth-child(12) { --i: 12; }

@keyframes particleAnimation {
  0% {
    opacity: 0;
    transform: rotate(calc(var(--i) * 30deg)) translateX(0) scale(0);
  }
  50% {
    opacity: 1;
    transform: rotate(calc(var(--i) * 30deg)) translateX(150px) scale(1);
  }
  100% {
    opacity: 0;
    transform: rotate(calc(var(--i) * 30deg)) translateX(200px) scale(0.5);
  }
}

/* 响应式离线弹窗 */
@media (max-width: 480px) {
  .offline-reward-modal {
    padding: 20px;
    margin: 20px;
  }

  .reward-stats {
    flex-direction: column;
    gap: 15px;
  }

  .items-grid {
    grid-template-columns: repeat(auto-fit, minmax(60px, 1fr));
  }
}
</style>
