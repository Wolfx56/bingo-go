<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type BingoEvent } from '../api'
import EventList from '../components/EventList.vue'

const events = ref<BingoEvent[]>([])
const loading = ref(true)
const loadError = ref('')

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    events.value = await api.history()
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : 'Chargement impossible.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page page--narrow">
    <h1 class="page-title">Historique</h1>
    <p class="muted intro">Toutes les validations, tous bingos confondus, de la plus récente à la plus ancienne.</p>

    <div v-if="loading" class="center-state">
      <v-progress-circular indeterminate color="primary" aria-label="Chargement" />
    </div>

    <v-alert v-else-if="loadError" type="error" variant="tonal" :text="loadError">
      <template #append>
        <v-btn variant="text" @click="load">Réessayer</v-btn>
      </template>
    </v-alert>

    <v-empty-state
      v-else-if="!events.length"
      headline="Rien à afficher"
      text="L’historique se remplit dès que vous créez un bingo et validez des cases."
    />

    <v-card v-else variant="flat" border class="panel">
      <EventList :events="events" show-bingo />
    </v-card>
  </div>
</template>

<style scoped>
.intro {
  margin: 6px 0 24px;
}

.panel {
  padding: 20px;
}
</style>
