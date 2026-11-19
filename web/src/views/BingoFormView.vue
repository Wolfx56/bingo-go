<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { mdiArrowLeft, mdiCheck, mdiFormatListBulleted, mdiShuffleVariant } from '@mdi/js'
import { api, type BingoInput } from '../api'
import { notify, notifyError } from '../notify'

const SIZES = [3, 4, 5, 6]
const MAX_CELLS = 36
const MAX_LABEL = 200

const route = useRoute()
const router = useRouter()
const editId = route.params.id ? Number(route.params.id) : null

const title = ref('')
const description = ref('')
const size = ref(5)
// 36-label buffer: changing the grid size never loses what was typed.
const labels = ref<string[]>(Array(MAX_CELLS).fill(''))
const checked = ref<boolean[]>([])

const loading = ref(editId !== null)
const saving = ref(false)
const submitted = ref(false)

const count = computed(() => size.value * size.value)
const visible = computed(() => labels.value.slice(0, count.value))
const filled = computed(() => visible.value.filter((l) => l.trim()).length)
const backTo = editId === null ? '/' : `/bingos/${editId}`

const snapshot = () =>
  JSON.stringify([title.value, description.value, size.value, visible.value])
let pristine = snapshot()

onMounted(async () => {
  if (editId === null) return
  try {
    const b = await api.getBingo(editId)
    if (b.archivedAt) {
      notify('Ce bingo est archivé : désarchivez-le pour le modifier.', 'error')
      router.replace(backTo)
      return
    }
    title.value = b.title
    description.value = b.description
    size.value = b.size
    b.cells.forEach((c) => {
      labels.value[c.position] = c.label
      checked.value[c.position] = !!c.checkedAt
    })
    pristine = snapshot()
    loading.value = false
  } catch (e) {
    notifyError(e)
    router.replace('/')
  }
})

onBeforeRouteLeave(() => {
  if (snapshot() !== pristine && !window.confirm('Quitter sans enregistrer vos modifications ?')) {
    return false
  }
})

// --- Grid input ---
const cellInputs = ref<HTMLTextAreaElement[]>([])

function focusCell(i: number) {
  cellInputs.value[i]?.focus()
}

function shuffle() {
  const items = visible.value
  for (let i = items.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[items[i], items[j]] = [items[j]!, items[i]!]
  }
  labels.value.splice(0, items.length, ...items)
}

// --- List input (one cell per line) ---
const listDialog = ref(false)
const listText = ref('')
const listCount = computed(() => listText.value.split('\n').filter((l) => l.trim()).length)

function openList() {
  listText.value = visible.value.join('\n').replace(/\n+$/, '')
  listDialog.value = true
}

function applyList() {
  const lines = listText.value
    .split('\n')
    .map((l) => l.trim().slice(0, MAX_LABEL))
    .filter(Boolean)
    .slice(0, count.value)
  const padded = [...lines, ...Array(count.value - lines.length).fill('')]
  labels.value.splice(0, count.value, ...padded)
  listDialog.value = false
}

// --- Saving ---
async function submit() {
  submitted.value = true
  if (!title.value.trim()) {
    notify('Donnez un titre à votre bingo.', 'error')
    return
  }
  const firstEmpty = visible.value.findIndex((l) => !l.trim())
  if (firstEmpty !== -1) {
    notify(`Il reste ${count.value - filled.value} case(s) à remplir.`, 'error')
    await nextTick()
    focusCell(firstEmpty)
    return
  }

  const input: BingoInput = {
    title: title.value,
    description: description.value,
    size: size.value,
    cells: visible.value,
  }
  saving.value = true
  try {
    const saved = editId === null ? await api.createBingo(input) : await api.updateBingo(editId, input)
    pristine = snapshot()
    notify(editId === null ? 'Bingo créé.' : 'Modifications enregistrées.', 'success')
    router.push(`/bingos/${saved.id}`)
  } catch (e) {
    notifyError(e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="page page--narrow">
    <v-btn :to="backTo" variant="text" :prepend-icon="mdiArrowLeft" class="back">
      {{ editId === null ? 'Tous les bingos' : 'Retour au bingo' }}
    </v-btn>
    <h1 class="page-title">{{ editId === null ? 'Nouveau bingo' : 'Modifier le bingo' }}</h1>

    <div v-if="loading" class="center-state">
      <v-progress-circular indeterminate color="primary" aria-label="Chargement" />
    </div>

    <form v-else class="form" novalidate @submit.prevent="submit">
      <v-text-field
        v-model="title"
        label="Titre"
        placeholder="Ex. : Bingo des vacances"
        maxlength="120"
        :error-messages="submitted && !title.trim() ? 'Le titre est obligatoire.' : ''"
        autofocus
      />
      <v-textarea
        v-model="description"
        label="Description (facultatif)"
        rows="2"
        auto-grow
        maxlength="500"
        counter
      />

      <fieldset class="group">
        <legend class="section-title">Taille de la grille</legend>
        <v-btn-toggle
          v-model="size"
          mandatory
          divided
          variant="outlined"
          color="primary"
          :disabled="editId !== null"
        >
          <v-btn v-for="n in SIZES" :key="n" :value="n">{{ n }} × {{ n }}</v-btn>
        </v-btn-toggle>
        <p v-if="editId !== null" class="muted hint">
          La taille ne peut plus être changée une fois le bingo créé.
        </p>
      </fieldset>

      <fieldset class="group">
        <legend class="section-title">Cases</legend>
        <div class="toolbar">
          <span class="muted counter" :class="{ 'counter--done': filled === count }">
            {{ filled }} / {{ count }} remplies
          </span>
          <v-spacer />
          <v-btn variant="tonal" size="small" :prepend-icon="mdiFormatListBulleted" @click="openList">
            Saisir en liste
          </v-btn>
          <v-btn
            v-if="editId === null"
            variant="tonal"
            size="small"
            :prepend-icon="mdiShuffleVariant"
            :disabled="filled < 2"
            @click="shuffle"
          >
            Mélanger
          </v-btn>
        </div>

        <div class="grid" :style="{ '--n': size }">
          <label
            v-for="(_, i) in visible"
            :key="i"
            class="cell"
            :class="{
              'cell--error': submitted && !labels[i]!.trim(),
              'cell--checked': checked[i],
            }"
          >
            <span class="sr-only">Case {{ i + 1 }}{{ checked[i] ? ' (déjà validée)' : '' }}</span>
            <textarea
              :ref="(el) => (cellInputs[i] = el as HTMLTextAreaElement)"
              v-model="labels[i]"
              class="cell__input"
              :placeholder="String(i + 1)"
              :maxlength="MAX_LABEL"
              rows="3"
              enterkeyhint="next"
              @keydown.enter.prevent="focusCell(i + 1)"
            />
            <v-icon v-if="checked[i]" :icon="mdiCheck" size="14" class="cell__check" />
          </label>
        </div>
        <p class="muted hint">
          Entrée passe à la case suivante.
          <template v-if="checked.some(Boolean)">Les cases cochées sont déjà validées et le restent.</template>
        </p>
      </fieldset>

      <div class="actions">
        <v-btn :to="backTo" variant="text" size="large">Annuler</v-btn>
        <v-btn type="submit" color="primary" variant="flat" size="large" :loading="saving" :prepend-icon="mdiCheck">
          {{ editId === null ? 'Créer le bingo' : 'Enregistrer' }}
        </v-btn>
      </div>
    </form>

    <v-dialog v-model="listDialog" max-width="560">
      <v-card>
        <v-card-title class="dialog-title">Saisir en liste</v-card-title>
        <v-card-text>
          <p class="muted mb">Une case par ligne, dans l’ordre de la grille (de gauche à droite, de haut en bas).</p>
          <v-textarea
            v-model="listText"
            rows="10"
            autofocus
            hide-details
            :placeholder="'Première case\nDeuxième case\n…'"
          />
          <p class="hint" :class="listCount > count ? 'over' : 'muted'">
            {{ listCount }} / {{ count }} lignes<template v-if="listCount > count">
              —
              {{
                listCount - count === 1
                  ? 'la dernière sera ignorée'
                  : `les ${listCount - count} dernières seront ignorées`
              }}</template>
          </p>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="listDialog = false">Annuler</v-btn>
          <v-btn color="primary" variant="flat" @click="applyList">Appliquer</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.back {
  margin: 0 0 8px -12px;
}

.form {
  display: grid;
  gap: 12px;
  margin-top: 24px;
}

.group {
  border: 0;
  padding: 0;
  margin: 0 0 12px;
  min-width: 0;
}

.group legend {
  margin-bottom: 10px;
}

.hint {
  margin-top: 8px;
  font-size: 0.8125rem;
}

.over {
  color: rgb(var(--v-theme-error));
}

.mb {
  margin-bottom: 12px;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.counter {
  font-size: 0.875rem;
  font-variant-numeric: tabular-nums;
}

.counter--done {
  color: rgb(var(--v-theme-success));
  font-weight: 600;
}

.grid {
  display: grid;
  grid-template-columns: repeat(var(--n), minmax(0, 1fr));
  gap: clamp(4px, 1.2vw, 8px);
  /* Keeps cells from getting oversized on small grids. */
  max-width: calc(var(--n) * 150px);
}

.cell {
  position: relative;
  aspect-ratio: 1;
  container-type: inline-size;
  display: grid;
  background: rgb(var(--v-theme-surface));
  border: 1px solid rgba(var(--v-theme-on-surface), 0.22);
  border-radius: clamp(8px, 1.6vw, 14px);
  overflow: hidden;
  cursor: text;
  transition: border-color 0.15s;
}

.cell:focus-within {
  border-color: rgb(var(--v-theme-primary));
  box-shadow: 0 0 0 1px rgb(var(--v-theme-primary));
}

.cell--error {
  border-color: rgb(var(--v-theme-error));
  background: rgba(var(--v-theme-error), 0.06);
}

.cell__input {
  width: 100%;
  height: 100%;
  padding: 8cqi;
  border: 0;
  outline: 0;
  resize: none;
  background: transparent;
  color: inherit;
  font: inherit;
  font-size: clamp(0.6875rem, 10cqi, 1rem);
  line-height: 1.25;
  text-align: center;
}

.cell__input::placeholder {
  color: rgba(var(--v-theme-on-surface), 0.35);
}

/* Centers the text vertically where the browser supports it. */
@supports (field-sizing: content) {
  .cell {
    align-items: center;
  }

  .cell__input {
    field-sizing: content;
    height: auto;
    max-height: 100cqi;
  }
}

.cell__check {
  position: absolute;
  top: 4px;
  right: 4px;
  color: rgb(var(--v-theme-primary));
  pointer-events: none;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}

</style>
