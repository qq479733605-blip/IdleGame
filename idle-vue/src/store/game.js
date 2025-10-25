import { defineStore } from "pinia";
import { gameConfig } from "../config";

export const useGameStore = defineStore("game", {
  state: () => ({
    // 玩家数据
    exp: 0,
    seqLevels: {},
    bag: {},
    equipment: {},

    // 当前运行状态
    isRunning: false,
    currentSeqId: "",
    currentSeqLevel: 0,
    currentSeqExp: 0,
    activeSubProject: "",

    // 配置数据
    sequences: [],
    equipmentCatalog: {},
    equipmentBonus: {
      exp_multiplier: 0,
      gain_multiplier: 0,
      rare_chance_bonus: 0
    }
  }),

  getters: {
    // 获取序列等级
    getSequenceLevel: (state) => (seqId) => {
      return state.seqLevels[seqId] || 1;
    },

    // 检查子项目是否解锁
    isSubProjectUnlocked: (state) => (seqId, subProject) => {
      const level = state.seqLevels[seqId] || 1;
      return level >= subProject.unlock_level;
    },

    // 获取当前序列配置
    currentSequenceConfig: (state) => {
      if (!state.currentSeqId) return null;
      return gameConfig.getSequenceConfig(state.currentSeqId);
    },

    // 获取当前子项目配置
    currentSubProjectConfig: (state) => {
      if (!state.currentSeqId || !state.activeSubProject) return null;
      return gameConfig.getSubProject(state.currentSeqId, state.activeSubProject);
    }
  },

  actions: {
    // 初始化游戏配置
    initializeGame() {
      this.sequences = gameConfig.getSequenceSummaries();
      this.equipmentCatalog = gameConfig.getEquipmentCatalog();
    },

    // 更新玩家数据
    updatePlayerData(data) {
      this.exp = data.exp || 0;
      this.seqLevels = data.seq_levels || {};
      this.bag = data.bag || {};
      this.equipment = data.equipment || {};
      this.equipmentBonus = data.equipment_bonus || {
        exp_multiplier: 0,
        gain_multiplier: 0,
        rare_chance_bonus: 0
      };
    },

    // 更新序列运行状态
    updateSequenceStatus(data) {
      this.isRunning = data.is_running || false;
      this.currentSeqId = data.seq_id || "";
      this.currentSeqLevel = data.seq_level || 0;
      this.currentSeqExp = data.current_seq_exp || 0;
      this.activeSubProject = data.active_sub_project || "";
    },

    // 更新背包
    updateBag(bag) {
      this.bag = bag;
    },

    // 更新经验
    updateExp(exp) {
      this.exp = exp;
    },

    // 更新序列等级
    updateSequenceLevel(seqId, level) {
      this.seqLevels[seqId] = level;
    },

    // 更新装备
    updateEquipment(equipment) {
      this.equipment = equipment;
      // 重新计算装备加成
      this.calculateEquipmentBonus();
    },

    // 计算装备加成
    calculateEquipmentBonus() {
      let totalBonus = {
        exp_multiplier: 0,
        gain_multiplier: 0,
        rare_chance_bonus: 0
      };

      for (const [slot, equipmentData] of Object.entries(this.equipment)) {
        if (equipmentData && equipmentData.item_id && this.equipmentCatalog[equipmentData.item_id]) {
          const item = this.equipmentCatalog[equipmentData.item_id];
          const attrs = item.attributes;

          totalBonus.exp_multiplier += attrs.exp_multiplier || 0;
          totalBonus.gain_multiplier += attrs.gain_multiplier || 0;
          totalBonus.rare_chance_bonus += attrs.rare_chance_bonus || 0;
        }
      }

      this.equipmentBonus = totalBonus;
    },

    // 登出时清空数据
    logout() {
      this.exp = 0;
      this.seqLevels = {};
      this.bag = {};
      this.equipment = {};
      this.isRunning = false;
      this.currentSeqId = "";
      this.currentSeqLevel = 0;
      this.activeSubProject = "";
      this.equipmentBonus = {
        exp_multiplier: 0,
        gain_multiplier: 0,
        rare_chance_bonus: 0
      };
    }
  }
});