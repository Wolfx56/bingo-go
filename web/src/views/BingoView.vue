<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  mdiArchiveArrowDownOutline,
  mdiArchiveArrowUpOutline,
  mdiArrowLeft,
  mdiCheck,
  mdiCheckBold,
  mdiContentCopy,
  mdiDotsVertical,
  mdiPartyPopper,
  mdiPencilOutline,
  mdiTrashCanOutline,
  mdiUndoVariant,
} from '@mdi/js'
import { api, ApiError, type Bingo, type BingoEvent, type Cell } from '../api'
import { notify, notifyError } from '../notify'
import { formatDateTime } from '../format'
import ConfettiBurst from '../components/ConfettiBurst.vue'
import EventList from '../components/EventList.vue'

const route = useRoute()
const router = useRouter()
const id = Number(route.params.id)

const bingo = ref<Bingo | null>(null)
const events = ref<BingoEvent[]>([])
const loading = ref(true)
const missing = ref(false)
const loadError = ref('')

const archived = computed(() => !!bingo.value?.archivedAt)
const full = computed(() => !!bingo.value && bingo.value.checked === bingo.value.total)
const percent = computed(() => (bingo.value ? (bingo.value.checked / bingo.value.total) * 100 : 0))
const lineCells = computed(() => new Set(bingo.value?.lines.flatMap((l) => l.cells) ?? []))

async function refreshEvents() {
  try {
    events.value = await api.bingoHistory(id)
  } catch {
    // history is secondary: the grid stays usable without it
  }
}

function apply(updated: Bingo) {
  bingo.value = updated
  document.title = `${updated.title} · Bingo`
  refreshEvents()
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    apply(await api.getBingo(id))
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) {
      missing.value = true
      document.title = 'Bingo introuvable · Bingo'
    } else {
      loadError.value = e instanceof Error ? e.message : 'Chargement impossible.'
    }
  } finally {
    loading.value = false
  }
}

onMounted(load)

// --- Cell validation ---
const cellDialog = ref(false)
const selected = ref<Cell | null>(null)
const note = ref('')
const saving = ref(false)

const selectedPlace = computed(() => {
  if (!bingo.value || !selected.value) return ''
  const pos = selected.value.position
  return `Ligne ${Math.floor(pos / bingo.value.size) + 1}, colonne ${(pos % bingo.value.size) + 1}`
})

function openCell(cell: Cell) {
  selected.value = cell
  note.value = cell.note
  cellDialog.value = true
}

async function validate() {
  if (!bingo.value || !selected.value) return
  const wasChecked = !!selected.value.checkedAt
  const known = new Set(bingo.value.lines.map((l) => l.name))
  saving.value = true
  try {
    const updated = await api.validateCell(id, selected.value.id, note.value)
    apply(updated)
    cellDialog.value = false
    const won = updated.lines.filter((l) => !known.has(l.name)).map((l) => l.name)
    if (won.length) celebrate(won, updated.checked === updated.total)
    else notify(wasChecked ? 'Note enregistrée.' : 'Case validée et enregistrée.', 'success')
  } catch (e) {
    notifyError(e)
  } finally {
    saving.value = false
  }
}

async function unvalidate() {
  if (!selected.value) return
  saving.value = true
  try {
    apply(await api.unvalidateCell(id, selected.value.id))
    cellDialog.value = false
    notify('Validation annulée.')
  } catch (e) {
    notifyError(e)
  } finally {
    saving.value = false
  }
}

// --- Celebration when a line is completed ---
const celebration = ref<{ lines: string[]; full: boolean } | null>(null)
const celebrating = ref(false)
const confettiRun = ref(0)
let confettiTimer: number | undefined

function celebrate(lines: string[], isFull: boolean) {
  celebration.value = { lines, full: isFull }
  celebrating.value = true
  confettiRun.value++
  window.clearTimeout(confettiTimer)
  confettiTimer = window.setTimeout(() => (confettiRun.value = 0), 4000)
}

onBeforeUnmount(() => window.clearTimeout(confettiTimer))

// --- Bingo actions ---
const busy = ref(false)

async function setArchived(value: boolean) {
  busy.value = true
  try {
    apply(await (value ? api.archiveBingo(id) : api.unarchiveBingo(id)))
    notify(value ? 'Bingo archivé.' : 'Bingo désarchivé.')
  } catch (e) {
    notifyError(e)
  } finally {
    busy.value = false
  }
}

const duplicateDialog = ref(false)
const duplicateTitle = ref('')
const duplicateShuffle = ref(true)

function openDuplicate() {
  duplicateTitle.value = bingo.value?.title ?? ''
  duplicateDialog.value = true
}

async function duplicate() {
  busy.value = true
  try {
    const copy = await api.duplicateBingo(id, duplicateTitle.value, duplicateShuffle.value)
    duplicateDialog.value = false
    notify('Nouvelle partie créée.', 'success')
    router.push(`/bingos/${copy.id}`)
  } catch (e) {
    notifyError(e)
  } finally {
    busy.value = false
  }
}

const deleteDialog = ref(false)

async function remove() {
  busy.value = true
  try {
    await api.deleteBingo(id)
    deleteDialog.value = false
    notify('Bingo supprimé.')
    router.replace('/')
  } catch (e) {
    notifyError(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="page">
    <div v-if="loading" class="center-state">
      <v-progress-circular indeterminate color="primary" aria-label="Chargement" />
    </div>

    <v-empty-state
      v-else-if="missing"
      headline="Bingo introuvable"
      text="Ce bingo n’existe pas ou a été supprimé."
    >
      <template #actions>
        <v-btn to="/" color="primary" variant="flat">Retour aux bingos</v-btn>
      </template>
    </v-empty-state>

    <v-alert v-else-if="loadError" type="error" variant="tonal" :text="loadError">
      <template #append>
        <v-btn variant="text" @click="load">Réessayer</v-btn>
      </template>
    </v-alert>

    <template v-else-if="bingo">
      <v-btn to="/" variant="text" :prepend-icon="mdiArrowLeft" class="back">Tous les bingos</v-btn>

      <header class="head">
        <div class="head__text">
          <h1 class="page-title">{{ bingo.title }}</h1>
          <p v-if="bingo.description" class="muted head__desc">{{ bingo.description }}</p>
        </div>
        <div class="head__actions">
          <v-btn
            v-if="!archived"
            :to="`/bingos/${bingo.id}/edit`"
            variant="tonal"
            :prepend-icon="mdiPencilOutline"
          >
            Modifier
          </v-btn>
          <v-menu location="bottom end">
            <template #activator="{ props }">
              <v-btn v-bind="props" :icon="mdiDotsVertical" variant="text" aria-label="Autres actions" />
            </template>
            <v-list density="comfortable" min-width="230">
              <v-list-item :prepend-icon="mdiContentCopy" title="Rejouer cette grille" @click="openDuplicate" />
              <v-list-item
                v-if="archived"
                :prepend-icon="mdiArchiveArrowUpOutline"
                title="Désarchiver"
                @click="setArchived(false)"
              />
              <v-list-item
                v-else
                :prepend-icon="mdiArchiveArrowDownOutline"
                title="Archiver"
                @click="setArchived(true)"
              />
              <v-divider class="my" />
              <v-list-item
                :prepend-icon="mdiTrashCanOutline"
                title="Supprimer…"
                base-color="error"
                @click="deleteDialog = true"
              />
            </v-list>
          </v-menu>
        </div>
      </header>

      <v-alert v-if="archived" type="info" variant="tonal" class="archived">
        Archivé le {{ formatDateTime(bingo.archivedAt!) }} — ce bingo est en lecture seule.
        <template #append>
          <v-btn variant="text" :loading="busy" @click="setArchived(false)">Désarchiver</v-btn>
        </template>
      </v-alert>

      <div class="layout">
        <div class="grid" :style="{ '--n': bingo.size }" role="group" aria-label="Grille de bingo">
          <button
            v-for="cell in bingo.cells"
            :key="cell.id"
            type="button"
            class="cell"
            :class="{
              'cell--checked': cell.checkedAt,
              'cell--line': lineCells.has(cell.position),
            }"
            :aria-pressed="!!cell.checkedAt"
            @click="openCell(cell)"
          >
            <span class="cell__inner">
              <span class="cell__label">{{ cell.label }}</span>
            </span>
            <v-icon v-if="cell.checkedAt" :icon="mdiCheckBold" class="cell__check" />
          </button>
        </div>

        <aside class="side">
          <v-card variant="flat" border class="panel">
            <h2 class="section-title">Progression</h2>
            <p class="score">
              <strong>{{ bingo.checked }}</strong>
              <span class="muted"> / {{ bingo.total }} cases</span>
            </p>
            <v-progress-linear
              :model-value="percent"
              color="primary"
              height="8"
              rounded
              :aria-label="`Progression : ${bingo.checked} cases sur ${bingo.total}`"
            />
            <div v-if="bingo.lines.length" class="lines">
              <v-chip v-if="full" color="secondary" variant="flat" size="small" :prepend-icon="mdiPartyPopper">
                Grille complète
              </v-chip>
              <v-chip v-for="l in bingo.lines" :key="l.name" color="secondary" variant="flat" size="small">
                {{ l.name }}
              </v-chip>
            </div>
            <p v-else class="muted hint">Complétez une ligne, une colonne ou une diagonale pour faire bingo.</p>
          </v-card>

          <v-card variant="flat" border class="panel">
            <h2 class="section-title">Historique</h2>
            <div class="history">
              <EventList :events="events" />
            </div>
          </v-card>
        </aside>
      </div>

      <!-- Cell validation -->
      <v-dialog v-model="cellDialog">
        <v-card v-if="selected">
          <v-card-text class="cell-dialog">
            <p class="muted cell-dialog__place">{{ selectedPlace }}</p>
            <h2 class="cell-dialog__label">{{ selected.label }}</h2>

            <p v-if="selected.checkedAt" class="cell-dialog__status">
              <v-icon :icon="mdiCheck" size="18" />
              Validée le {{ formatDateTime(selected.checkedAt) }}
            </p>

            <template v-if="archived">
              <p v-if="selected.note" class="cell-dialog__note">{{ selected.note }}</p>
              <p v-else-if="!selected.checkedAt" class="muted">Case non validée.</p>
            </template>
            <v-textarea
              v-else
              v-model="note"
              label="Note (facultatif)"
              rows="2"
              auto-grow
              maxlength="1000"
              hide-details
              class="cell-dialog__field"
            />
          </v-card-text>
          <v-card-actions class="cell-dialog__actions">
            <v-btn
              v-if="selected.checkedAt && !archived"
              color="error"
              variant="text"
              :prepend-icon="mdiUndoVariant"
              :disabled="saving"
              @click="unvalidate"
            >
              Annuler la validation
            </v-btn>
            <v-spacer />
            <v-btn variant="text" @click="cellDialog = false">Fermer</v-btn>
            <v-btn
              v-if="!archived && !selected.checkedAt"
              color="primary"
              variant="flat"
              :prepend-icon="mdiCheck"
              :loading="saving"
              @click="validate"
            >
              Valider
            </v-btn>
            <v-btn
              v-else-if="!archived && note.trim() !== selected.note"
              color="primary"
              variant="flat"
              :loading="saving"
              @click="validate"
            >
              Enregistrer la note
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-dialog>

      <!-- Replay -->
      <v-dialog v-model="duplicateDialog">
        <v-card>
          <v-card-title class="dialog-title">Rejouer cette grille</v-card-title>
          <v-card-text>
            <p class="muted mb">
              Crée un nouveau bingo vierge avec les mêmes cases. Celui-ci est conservé tel quel.
            </p>
            <v-text-field v-model="duplicateTitle" label="Titre de la nouvelle partie" maxlength="120" />
            <v-switch v-model="duplicateShuffle" label="Mélanger les cases" color="primary" hide-details />
          </v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn variant="text" @click="duplicateDialog = false">Annuler</v-btn>
            <v-btn color="primary" variant="flat" :loading="busy" @click="duplicate">Créer</v-btn>
          </v-card-actions>
        </v-card>
      </v-dialog>

      <!-- Deletion -->
      <v-dialog v-model="deleteDialog">
        <v-card>
          <v-card-title class="dialog-title">Supprimer ce bingo ?</v-card-title>
          <v-card-text>
            « {{ bingo.title }} », ses cases et tout son historique seront définitivement supprimés. Pour le
            garder en lecture seule, archivez-le plutôt.
          </v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn variant="text" @click="deleteDialog = false">Annuler</v-btn>
            <v-btn color="error" variant="flat" :loading="busy" @click="remove">Supprimer</v-btn>
          </v-card-actions>
        </v-card>
      </v-dialog>

      <v-snackbar v-model="celebrating" location="top" color="secondary" :timeout="6000">
        <div v-if="celebration" class="bingo-toast" role="status">
          <v-icon :icon="mdiPartyPopper" size="28" />
          <div>
            <p class="bingo-toast__title">{{ celebration.full ? 'Grille complète !' : 'Bingo !' }}</p>
            <p>{{ celebration.lines.join(', ') }} {{ celebration.lines.length > 1 ? 'complétées' : 'complétée' }}</p>
          </div>
        </div>
      </v-snackbar>
      <ConfettiBurst v-if="confettiRun" :key="confettiRun" />
    </template>
  </div>
</template>

<style scoped>
.back {
  margin: 0 0 8px -12px;
}

.head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px 24px;
  margin-bottom: 24px;
}

.head__text {
  min-width: 0;
  flex: 1 1 280px;
}

.head__desc {
  margin-top: 6px;
  max-width: 70ch;
  white-space: pre-line;
  overflow-wrap: anywhere;
}

.head__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.my {
  margin: 4px 0;
}

.archived {
  margin-bottom: 24px;
}

.layout {
  display: grid;
  gap: 24px;
}

@media (min-width: 960px) {
  .layout {
    grid-template-columns: minmax(0, 1fr) 340px;
    align-items: start;
  }
}

/* --- Grid --- */
.grid {
  display: grid;
  grid-template-columns: repeat(var(--n), minmax(0, 1fr));
  gap: clamp(4px, 1.2vw, 10px);
  width: 100%;
  /* Keeps the whole grid on screen on desktop. */
  max-width: max(340px, min(700px, 100dvh - 230px));
  margin: 0 auto;
}

.cell {
  position: relative;
  aspect-ratio: 1;
  container-type: inline-size;
  background: rgb(var(--v-theme-surface));
  color: rgb(var(--v-theme-on-surface));
  border: 1px solid rgba(var(--v-theme-on-surface), 0.16);
  border-radius: clamp(8px, 1.6vw, 16px);
  cursor: pointer;
  transition:
    background-color 0.2s,
    border-color 0.2s,
    transform 0.1s;
}

.cell:hover {
  border-color: rgb(var(--v-theme-primary));
}

.cell:active {
  transform: scale(0.97);
}

.cell:focus-visible {
  outline: 3px solid rgb(var(--v-theme-primary));
  outline-offset: 2px;
}

.cell--checked {
  background: rgb(var(--v-theme-primary));
  border-color: rgb(var(--v-theme-primary));
  color: rgb(var(--v-theme-on-primary));
}

.cell--line {
  background: rgb(var(--v-theme-secondary));
  border-color: rgb(var(--v-theme-secondary));
  color: rgb(var(--v-theme-on-secondary));
}

.cell__inner {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 8cqi;
  overflow: hidden;
}

.cell__label {
  font-size: clamp(0.6875rem, 10.5cqi, 1.0625rem);
  font-weight: 500;
  line-height: 1.2;
  text-align: center;
  overflow-wrap: anywhere;
  hyphens: auto;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 4;
  overflow: hidden;
}

.cell__check {
  position: absolute;
  top: 5cqi;
  right: 5cqi;
  font-size: clamp(11px, 13cqi, 18px);
  opacity: 0.75;
}

@container (max-width: 68px) {
  .cell__inner {
    padding: 3px;
  }

  .cell__label {
    font-size: 0.625rem;
    -webkit-line-clamp: 3;
  }

  .cell__check {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .cell {
    transition: none;
  }
}

/* --- Side column --- */
.side {
  display: grid;
  gap: 16px;
}

.panel {
  padding: 20px;
}

.score {
  margin: 6px 0 10px;
  font-variant-numeric: tabular-nums;
}

.score strong {
  font-family: var(--font-display);
  font-size: 2rem;
  font-weight: 800;
  line-height: 1;
}

.lines {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 14px;
}

.hint {
  margin-top: 12px;
  font-size: 0.875rem;
}

.history {
  margin-top: 14px;
  max-height: 420px;
  overflow-y: auto;
}

/* --- Dialogs --- */

.mb {
  margin-bottom: 16px;
}

.cell-dialog {
  padding-top: 24px;
}

.cell-dialog__place {
  font-size: 0.8125rem;
}

.cell-dialog__label {
  margin: 2px 0 12px;
  font-family: var(--font-display);
  font-size: 1.35rem;
  font-weight: 700;
  line-height: 1.25;
  overflow-wrap: anywhere;
}

.cell-dialog__status {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  color: rgb(var(--v-theme-success));
  font-size: 0.9375rem;
  font-weight: 500;
}

.cell-dialog__note {
  padding-left: 12px;
  border-left: 2px solid rgba(var(--v-theme-on-surface), 0.2);
  white-space: pre-line;
  overflow-wrap: anywhere;
}

.cell-dialog__field {
  margin-top: 4px;
}

.cell-dialog__actions {
  flex-wrap: wrap;
}

.bingo-toast {
  display: flex;
  align-items: center;
  gap: 12px;
}

.bingo-toast__title {
  font-family: var(--font-display);
  font-size: 1.25rem;
  font-weight: 800;
  line-height: 1.2;
}
</style>
