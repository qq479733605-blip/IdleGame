<template>
  <div class="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
    <div class="bg-white p-6 rounded-lg w-96 max-h-[80vh] overflow-y-auto">
      <h2 class="text-lg font-bold mb-4">背包</h2>

      <!-- 装备区域 -->
      <div class="mb-6">
        <h3 class="text-md font-semibold mb-2 text-gray-700">装备</h3>
        <div class="grid grid-cols-2 gap-2 mb-4">
          <div v-for="(item, slot) in equipment" :key="slot"
               class="border rounded p-2 bg-gray-50"
               :class="getEquipmentQualityClass(item.quality)">
            <div class="text-xs text-gray-500">{{ getSlotName(slot) }}</div>
            <div class="font-medium text-sm">{{ item.name }}</div>
            <div class="text-xs text-gray-600">{{ getQualityText(item.quality) }}</div>
          </div>
          <div v-for="slot in getEmptySlots()" :key="'empty-' + slot"
               class="border rounded p-2 bg-gray-100 border-dashed">
            <div class="text-xs text-gray-400">{{ getSlotName(slot) }}</div>
            <div class="text-xs text-gray-400">空</div>
          </div>
        </div>
      </div>

      <!-- 物品区域 -->
      <div class="mb-4">
        <h3 class="text-md font-semibold mb-2 text-gray-700">物品</h3>
        <div class="space-y-2 max-h-60 overflow-y-auto">
          <div v-for="(count, itemId) in filteredBag" :key="itemId"
               class="flex justify-between items-center p-2 border rounded hover:bg-gray-50"
               :class="{ 'bg-blue-50 border-blue-200': isEquipment(itemId) }">
            <div class="flex-1">
              <div class="font-medium text-sm">
                {{ getItemName(itemId) }}
                <span v-if="isEquipment(itemId)" class="ml-2 text-xs bg-blue-100 text-blue-600 px-1 rounded">装备</span>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-sm font-semibold">{{ count }}</span>
              <button v-if="isEquipment(itemId) && canEquip(itemId)"
                      @click="equipItem(itemId)"
                      class="bg-blue-500 text-white px-2 py-1 rounded text-xs hover:bg-blue-600">
                装备
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 装备加成显示 -->
      <div class="mb-4 p-3 bg-gray-50 rounded">
        <h3 class="text-sm font-semibold mb-2 text-gray-700">装备加成</h3>
        <div class="text-xs space-y-1">
          <div v-if="equipmentBonus.gain_multiplier > 0">
            收益加成: +{{ (equipmentBonus.gain_multiplier * 100).toFixed(1) }}%
          </div>
          <div v-if="equipmentBonus.rare_chance_bonus > 0">
            稀有几率: +{{ (equipmentBonus.rare_chance_bonus * 100).toFixed(1) }}%
          </div>
          <div v-if="equipmentBonus.exp_multiplier > 0">
            经验加成: +{{ (equipmentBonus.exp_multiplier * 100).toFixed(1) }}%
          </div>
          <div v-if="!hasAnyBonus" class="text-gray-400">暂无装备加成</div>
        </div>
      </div>

      <button @click="$emit('close')" class="w-full bg-gray-700 text-white px-3 py-2 rounded hover:bg-gray-800">
        关闭
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useGameStore } from '../store/game'

const props = defineProps({
  bag: Object,
  equipment: Object,
  equipmentBonus: Object
})

const emit = defineEmits(['close', 'equip'])

const gameStore = useGameStore()

// 过滤掉已装备的物品
const filteredBag = computed(() => {
  const equippedItems = Object.values(props.equipment || {}).map(item => item.item_id)
  const filtered = {}
  for (const [itemId, count] of Object.entries(props.bag || {})) {
    if (!equippedItems.includes(itemId) || count > 1) {
      filtered[itemId] = equippedItems.includes(itemId) ? count - 1 : count
    }
  }
  return filtered
})

// 是否有装备加成
const hasAnyBonus = computed(() => {
  return props.equipmentBonus?.gain_multiplier > 0 ||
         props.equipmentBonus?.rare_chance_bonus > 0 ||
         props.equipmentBonus?.exp_multiplier > 0
})

// 获取空槽位
const getEmptySlots = () => {
  const allSlots = ['head', 'weapon', 'armor', 'hand', 'foot', 'relic']
  const usedSlots = Object.keys(props.equipment || {})
  return allSlots.filter(slot => !usedSlots.includes(slot))
}

// 获取槽位名称
const getSlotName = (slot) => {
  const slotNames = {
    head: '头部',
    weapon: '武器',
    armor: '护甲',
    hand: '手部',
    foot: '脚部',
    relic: '法宝'
  }
  return slotNames[slot] || slot
}

// 获取品质文本
const getQualityText = (quality) => {
  const qualityTexts = {
    common: '普通',
    uncommon: '优秀',
    rare: '稀有',
    epic: '史诗',
    legendary: '传说'
  }
  return qualityTexts[quality] || '普通'
}

// 获取装备品质样式
const getEquipmentQualityClass = (quality) => {
  const classes = {
    common: 'border-gray-300',
    uncommon: 'border-green-300',
    rare: 'border-blue-300',
    epic: 'border-purple-300',
    legendary: 'border-orange-300'
  }
  return classes[quality] || 'border-gray-300'
}

// 检查是否为装备
const isEquipment = (itemId) => {
  return gameStore.equipmentCatalog[itemId] !== undefined
}

// 获取物品名称
const getItemName = (itemId) => {
  const item = gameStore.equipmentCatalog[itemId]
  return item ? item.name : itemId
}

// 检查是否可以装备
const canEquip = (itemId) => {
  const item = gameStore.equipmentCatalog[itemId]
  if (!item) return false

  const slot = item.slot
  const currentEquipped = props.equipment?.[slot]

  // 如果槽位为空或者已有装备但背包里有更多数量，则可以装备
  return !currentEquipped || (props.bag[itemId] > 1)
}

// 装备物品
const equipItem = (itemId) => {
  emit('equip', itemId)
}
</script>
