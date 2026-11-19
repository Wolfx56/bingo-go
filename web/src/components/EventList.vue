<script setup lang="ts">
import { computed } from 'vue'
import {
  mdiArchiveArrowDownOutline,
  mdiArchiveArrowUpOutline,
  mdiCheck,
  mdiPartyPopper,
  mdiPlus,
  mdiUndoVariant,
  mdiVectorLine,
} from '@mdi/js'
import type { BingoEvent, EventKind } from '../api'
import { dayKey, dayLabel, formatTime } from '../format'

const props = defineProps<{ events: BingoEvent[]; showBingo?: boolean }>()

const styles: Record<EventKind, { icon: string; tone: string }> = {
  created: { icon: mdiPlus, tone: 'neutral' },
  validated: { icon: mdiCheck, tone: 'primary' },
  unvalidated: { icon: mdiUndoVariant, tone: 'neutral' },
  line: { icon: mdiVectorLine, tone: 'gold' },
  full: { icon: mdiPartyPopper, tone: 'gold' },
  archived: { icon: mdiArchiveArrowDownOutline, tone: 'neutral' },
  unarchived: { icon: mdiArchiveArrowUpOutline, tone: 'neutral' },
}

function describe(e: BingoEvent): string {
  switch (e.kind) {
    case 'created':
      return 'Bingo créé'
    case 'validated':
      return `« ${e.label} » validée`
    case 'unvalidated':
      return `Validation de « ${e.label} » annulée`
    case 'line':
      return `Bingo ! ${e.label} complète`
    case 'full':
      return 'Grille complète'
    case 'archived':
      return 'Bingo archivé'
    case 'unarchived':
      return 'Bingo désarchivé'
    default:
      return e.kind
  }
}

// Events arrive most recent first; group them by day.
const days = computed(() => {
  const groups: { key: string; label: string; events: BingoEvent[] }[] = []
  for (const e of props.events) {
    const key = dayKey(e.createdAt)
    let group = groups[groups.length - 1]
    if (!group || group.key !== key) {
      group = { key, label: dayLabel(e.createdAt), events: [] }
      groups.push(group)
    }
    group.events.push(e)
  }
  return groups
})
</script>

<template>
  <div class="events">
    <section v-for="day in days" :key="day.key">
      <h3 class="events__day muted">{{ day.label }}</h3>
      <ul class="events__list">
        <li v-for="e in day.events" :key="e.id" class="event">
          <span class="event__icon" :class="`event__icon--${(styles[e.kind] ?? styles.created).tone}`">
            <v-icon :icon="(styles[e.kind] ?? styles.created).icon" size="16" />
          </span>
          <div class="event__body">
            <p class="event__text">{{ describe(e) }}</p>
            <p v-if="e.note" class="event__note">{{ e.note }}</p>
            <p class="event__meta muted">
              <time :datetime="e.createdAt">{{ formatTime(e.createdAt) }}</time>
              <template v-if="showBingo">
                ·
                <router-link :to="`/bingos/${e.bingoId}`" class="event__link">{{ e.bingoTitle }}</router-link>
              </template>
            </p>
          </div>
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.events {
  display: grid;
  gap: 20px;
}

.events__day {
  font-size: 0.8125rem;
  font-weight: 600;
  margin-bottom: 8px;
}

.events__day::first-letter {
  text-transform: uppercase;
}

.events__list {
  list-style: none;
  padding: 0;
  display: grid;
  gap: 12px;
}

.event {
  display: flex;
  gap: 12px;
}

.event__icon {
  flex: none;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
}

.event__icon--neutral {
  background: rgba(var(--v-theme-on-surface), 0.08);
  color: rgba(var(--v-theme-on-surface), 0.7);
}

.event__icon--primary {
  background: rgb(var(--v-theme-primary));
  color: rgb(var(--v-theme-on-primary));
}

.event__icon--gold {
  background: rgb(var(--v-theme-secondary));
  color: rgb(var(--v-theme-on-secondary));
}

.event__body {
  min-width: 0;
  padding-top: 3px;
}

.event__text {
  font-size: 0.9375rem;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.event__note {
  margin-top: 4px;
  padding-left: 10px;
  border-left: 2px solid rgba(var(--v-theme-on-surface), 0.2);
  font-size: 0.875rem;
  white-space: pre-line;
  overflow-wrap: anywhere;
}

.event__meta {
  margin-top: 2px;
  font-size: 0.8125rem;
  font-variant-numeric: tabular-nums;
}

.event__link {
  color: rgb(var(--v-theme-primary));
  text-decoration: none;
  overflow-wrap: anywhere;
}

.event__link:hover {
  text-decoration: underline;
}
</style>
