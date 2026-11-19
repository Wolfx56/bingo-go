<script setup lang="ts">
import { computed } from 'vue'
import { mdiPartyPopper, mdiVectorLine } from '@mdi/js'
import type { BingoSummary } from '../api'
import { formatRelative, plural } from '../format'
import MiniGrid from './MiniGrid.vue'

const props = defineProps<{ bingo: BingoSummary }>()

const full = computed(() => props.bingo.checked === props.bingo.total)
const percent = computed(() => (props.bingo.checked / props.bingo.total) * 100)
</script>

<template>
  <v-card :to="`/bingos/${bingo.id}`" variant="flat" border class="card">
    <div class="card__body">
      <MiniGrid :size="bingo.size" :pattern="bingo.pattern" :lines="bingo.lines" />
      <div class="card__text">
        <h2 class="card__title">{{ bingo.title }}</h2>
        <p v-if="bingo.description" class="card__desc muted">{{ bingo.description }}</p>
        <div class="card__chips">
          <v-chip v-if="full" size="small" color="secondary" variant="flat" :prepend-icon="mdiPartyPopper">
            Grille complète
          </v-chip>
          <v-chip
            v-else-if="bingo.lines.length"
            size="small"
            color="secondary"
            variant="flat"
            :prepend-icon="mdiVectorLine"
          >
            {{ plural(bingo.lines.length, 'ligne', 'lignes') }}
          </v-chip>
        </div>
      </div>
    </div>
    <div class="card__footer muted">
      <span>{{ bingo.checked }} / {{ bingo.total }} cases</span>
      <span>{{ formatRelative(bingo.updatedAt) }}</span>
    </div>
    <v-progress-linear
      :model-value="percent"
      color="primary"
      height="4"
      :aria-label="`Progression : ${bingo.checked} cases sur ${bingo.total}`"
    />
  </v-card>
</template>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.card__body {
  display: flex;
  gap: 16px;
  padding: 18px 18px 12px;
  flex: 1;
}

.card__text {
  min-width: 0;
}

.card__title {
  font-family: var(--font-display);
  font-size: 1.1rem;
  font-weight: 700;
  line-height: 1.25;
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
}

.card__desc {
  margin-top: 2px;
  font-size: 0.875rem;
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
}

.card__chips {
  margin-top: 8px;
}

.card__chips:empty {
  display: none;
}

.card__footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 0 18px 12px;
  font-size: 0.8125rem;
  font-variant-numeric: tabular-nums;
}
</style>
