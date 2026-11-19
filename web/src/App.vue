<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useTheme } from 'vuetify'
import { mdiHistory, mdiViewGridOutline, mdiWeatherNight, mdiWhiteBalanceSunny } from '@mdi/js'
import { rememberTheme } from './theme'
import { snack } from './notify'

const route = useRoute()
const theme = useTheme()
const dark = computed(() => theme.global.current.value.dark)

function toggleTheme() {
  const next = dark.value ? 'light' : 'dark'
  theme.change(next)
  rememberTheme(next)
}
</script>

<template>
  <v-app>
    <v-app-bar flat border="b" density="comfortable">
      <router-link to="/" class="brand" aria-label="Bingo — accueil">
        <svg class="brand__logo" viewBox="0 0 32 32" aria-hidden="true">
          <rect width="32" height="32" rx="8" fill="rgb(var(--v-theme-primary))" />
          <g fill="rgb(var(--v-theme-on-primary))" fill-opacity=".35">
            <rect x="13" y="5" width="6" height="6" rx="1.5" />
            <rect x="21" y="5" width="6" height="6" rx="1.5" />
            <rect x="5" y="13" width="6" height="6" rx="1.5" />
            <rect x="21" y="13" width="6" height="6" rx="1.5" />
            <rect x="5" y="21" width="6" height="6" rx="1.5" />
            <rect x="13" y="21" width="6" height="6" rx="1.5" />
          </g>
          <g fill="rgb(var(--v-theme-secondary))">
            <rect x="5" y="5" width="6" height="6" rx="1.5" />
            <rect x="13" y="13" width="6" height="6" rx="1.5" />
            <rect x="21" y="21" width="6" height="6" rx="1.5" />
          </g>
        </svg>
        <span class="brand__name">Bingo</span>
      </router-link>

      <v-spacer />

      <v-btn to="/" exact variant="text" aria-label="Bingos">
        <v-icon :icon="mdiViewGridOutline" />
        <span class="nav-label">Bingos</span>
      </v-btn>
      <v-btn to="/historique" variant="text" aria-label="Historique">
        <v-icon :icon="mdiHistory" />
        <span class="nav-label">Historique</span>
      </v-btn>
      <v-btn
        :icon="dark ? mdiWhiteBalanceSunny : mdiWeatherNight"
        :aria-label="dark ? 'Passer au thème clair' : 'Passer au thème sombre'"
        variant="text"
        class="mr-1"
        @click="toggleTheme"
      />
    </v-app-bar>

    <v-main>
      <!-- La clé force le remontage quand on passe d'un bingo à un autre. -->
      <router-view :key="route.path" />
    </v-main>

    <v-snackbar v-model="snack.show" :color="snack.color" :timeout="3500">
      {{ snack.text }}
    </v-snackbar>
  </v-app>
</template>

<style scoped>
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: 16px;
  color: inherit;
  text-decoration: none;
  border-radius: 10px;
}

.brand__logo {
  width: 30px;
  height: 30px;
}

.brand__name {
  font-family: var(--font-display);
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: -0.01em;
}

.nav-label {
  margin-left: 8px;
}

@media (max-width: 520px) {
  .nav-label {
    display: none;
  }
}
</style>
