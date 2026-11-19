<script setup lang="ts">
import { computed } from 'vue'
import type { Line } from '../api'

const props = defineProps<{ size: number; pattern: boolean[]; lines: Line[] }>()

const inLine = computed(() => new Set(props.lines.flatMap((l) => l.cells)))
</script>

<template>
  <div class="mini" :style="{ '--n': size }" aria-hidden="true">
    <span
      v-for="(checked, i) in pattern"
      :key="i"
      class="mini__cell"
      :class="{ 'mini__cell--checked': checked, 'mini__cell--line': inLine.has(i) }"
    />
  </div>
</template>

<style scoped>
.mini {
  display: grid;
  grid-template-columns: repeat(var(--n), 1fr);
  gap: 3px;
  width: 76px;
  aspect-ratio: 1;
  flex: none;
}

.mini__cell {
  border-radius: 3px;
  background: rgba(var(--v-theme-on-surface), 0.1);
}

.mini__cell--checked {
  background: rgb(var(--v-theme-primary));
}

.mini__cell--line {
  background: rgb(var(--v-theme-secondary));
}
</style>
