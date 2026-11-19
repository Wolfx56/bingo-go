<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { mdiPlus } from '@mdi/js'
import { api, type BingoSummary } from '../api'
import BingoCard from '../components/BingoCard.vue'

const bingos = ref<BingoSummary[]>([])
const loading = ref(true)
const loadError = ref('')
const tab = ref<'active' | 'archived'>('active')

const active = computed(() => bingos.value.filter((b) => !b.archivedAt))
const archived = computed(() => bingos.value.filter((b) => b.archivedAt))
const shown = computed(() => (tab.value === 'active' ? active.value : archived.value))

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    bingos.value = await api.listBingos()
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : 'Chargement impossible.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <header class="head">
      <h1 class="page-title">Mes bingos</h1>
      <v-btn to="/new" color="primary" variant="flat" size="large" :prepend-icon="mdiPlus">
        Nouveau bingo
      </v-btn>
    </header>

    <div v-if="loading" class="center-state">
      <v-progress-circular indeterminate color="primary" aria-label="Chargement" />
    </div>

    <v-alert v-else-if="loadError" type="error" variant="tonal" :text="loadError">
      <template #append>
        <v-btn variant="text" @click="load">Réessayer</v-btn>
      </template>
    </v-alert>

    <v-empty-state
      v-else-if="!bingos.length"
      headline="Aucun bingo pour l’instant"
      text="Créez votre première grille : donnez-lui un titre, remplissez les cases, puis validez-les au fil du temps."
    >
      <template #actions>
        <v-btn to="/new" color="primary" variant="flat" :prepend-icon="mdiPlus">Créer un bingo</v-btn>
      </template>
    </v-empty-state>

    <template v-else>
      <v-tabs v-model="tab" color="primary" class="tabs">
        <v-tab value="active">En cours ({{ active.length }})</v-tab>
        <v-tab value="archived">Archivés ({{ archived.length }})</v-tab>
      </v-tabs>

      <p v-if="!shown.length" class="muted none">
        {{
          tab === 'active'
            ? 'Aucun bingo en cours. Créez-en un nouveau ou désarchivez-en un.'
            : 'Aucun bingo archivé. Archivez un bingo terminé pour le retrouver ici.'
        }}
      </p>
      <div v-else class="cards">
        <BingoCard v-for="b in shown" :key="b.id" :bingo="b" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.tabs {
  margin-bottom: 20px;
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 300px), 1fr));
  gap: 16px;
}

.none {
  padding: 32px 0;
}
</style>
