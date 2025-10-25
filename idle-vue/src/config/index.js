import sequencesData from './sequences.json'
import equipmentData from './equipment.json'

// 游戏配置管理
class GameConfig {
  constructor() {
    this.sequences = sequencesData
    this.equipment = equipmentData
  }

  // 获取所有序列概览
  getSequenceSummaries() {
    const briefs = []
    for (const [id, cfg] of Object.entries(this.sequences)) {
      const name = cfg.name || id

      const subs = (cfg.sub_projects || []).map(sp => ({
        id: sp.id,
        name: sp.name,
        unlockLevel: sp.unlock_level,
        description: sp.description,
        gainMultiplier: sp.gain_multiplier || 0,
        rareBonus: sp.rare_chance_bonus || 0,
        expMultiplier: sp.exp_multiplier || 0,
        intervalMod: sp.interval_modifier || 0
      }))

      briefs.push({
        id,
        name,
        tickInterval: cfg.tick_interval,
        subProjects: subs
      })
    }

    // 按 ID 排序，确保稳定顺序
    briefs.sort((a, b) => a.id.localeCompare(b.id))
    return briefs
  }

  // 获取序列配置
  getSequenceConfig(id) {
    return this.sequences[id] || null
  }

  // 获取子项目
  getSubProject(sequenceId, subProjectId) {
    const config = this.getSequenceConfig(sequenceId)
    if (!config || !config.sub_projects) return null

    return config.sub_projects.find(sp => sp.id === subProjectId) || null
  }

  // 获取装备配置
  getEquipmentConfig() {
    return this.equipment
  }

  // 获取装备目录
  getEquipmentCatalog() {
    const catalog = {}
    for (const [itemId, itemConfig] of Object.entries(this.equipment)) {
      catalog[itemId] = {
        itemId,
        name: itemConfig.name,
        description: itemConfig.description,
        quality: itemConfig.quality,
        slot: itemConfig.slot,
        attributes: itemConfig.attributes,
        enhancement: 0
      }
    }
    return catalog
  }

  // 获取有效时间间隔（考虑子项目修正）
  getEffectiveInterval(sequenceId, subProjectId) {
    const config = this.getSequenceConfig(sequenceId)
    if (!config) return 3000 // 默认3秒

    let base = config.tick_interval * 1000 // 转换为毫秒
    if (base <= 0) base = 3000

    if (subProjectId) {
      const subProject = this.getSubProject(sequenceId, subProjectId)
      if (subProject && subProject.interval_modifier > 0) {
        base = base * subProject.interval_modifier
      }
    }

    if (base < 500) base = 500 // 最小500ms
    return base
  }
}

// 导出单例实例
export const gameConfig = new GameConfig()
export default gameConfig