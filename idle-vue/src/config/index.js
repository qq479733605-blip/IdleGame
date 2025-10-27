// 游戏配置管理 - 简化版本
class GameConfig {
  constructor() {
    // 暂时为空，待后续实现
  }

  // 获取所有序列概览
  getSequenceSummaries() {
    return []
  }

  // 获取序列配置
  getSequenceConfig(id) {
    return null
  }

  // 获取子项目
  getSubProject(sequenceId, subProjectId) {
    return null
  }

  // 获取装备配置
  getEquipmentConfig() {
    return {}
  }

  // 获取装备目录
  getEquipmentCatalog() {
    return {}
  }

  // 获取有效时间间隔（考虑子项目修正）
  getEffectiveInterval(sequenceId, subProjectId) {
    return 3000 // 默认3秒
  }
}

// 导出单例实例
export const gameConfig = new GameConfig()
export default gameConfig