import { reactive } from 'vue'

// Global transient message, rendered by App.vue.
export const snack = reactive({ show: false, text: '', color: undefined as string | undefined })

export function notify(text: string, color?: string) {
  snack.text = text
  snack.color = color
  snack.show = true
}

export function notifyError(err: unknown) {
  notify(err instanceof Error ? err.message : 'Une erreur est survenue.', 'error')
}
