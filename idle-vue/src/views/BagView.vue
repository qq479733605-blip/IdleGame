<template>
  <div class="fixed inset-0 bg-black/60 flex items-center justify-center">
    <div class="bg-white p-6 rounded-lg w-80">
      <h2 class="text-lg font-bold mb-2">背包</h2>
      <div v-for="(v, k) in bag" :key="k" class="flex justify-between mb-1">
        <span>{{ k }}</span>
        <span>{{ v }}</span>
      </div>
      <button @click="$emit('close')" class="mt-3 bg-gray-700 text-white px-3 py-1 rounded">关闭</button>
    </div>
  </div>
</template>

<script setup>
defineProps({
  bag: Object
})
</script>
