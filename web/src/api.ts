export interface Cell {
  id: number
  position: number
  label: string
  checkedAt: string | null
  note: string
}

export interface Line {
  name: string
  cells: number[]
}

export interface BingoSummary {
  id: number
  title: string
  description: string
  size: number
  createdAt: string
  updatedAt: string
  archivedAt: string | null
  checked: number
  total: number
  pattern: boolean[]
  lines: Line[]
}

export interface Bingo extends BingoSummary {
  cells: Cell[]
}

export type EventKind =
  | 'created'
  | 'validated'
  | 'unvalidated'
  | 'line'
  | 'full'
  | 'archived'
  | 'unarchived'

export interface BingoEvent {
  id: number
  bingoId: number
  bingoTitle: string
  cellId: number | null
  kind: EventKind
  label: string
  note: string
  createdAt: string
}

export interface BingoInput {
  title: string
  description: string
  size: number
  cells: string[]
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response
  try {
    res = await fetch('/api' + path, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'Serveur injoignable. Vérifiez que Bingo est bien démarré.')
  }
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => null)
  if (!res.ok) throw new ApiError(res.status, data?.error ?? `Erreur ${res.status}`)
  return data as T
}

export const api = {
  listBingos: () => request<BingoSummary[]>('GET', '/bingos'),
  getBingo: (id: number) => request<Bingo>('GET', `/bingos/${id}`),
  createBingo: (input: BingoInput) => request<Bingo>('POST', '/bingos', input),
  updateBingo: (id: number, input: BingoInput) => request<Bingo>('PUT', `/bingos/${id}`, input),
  deleteBingo: (id: number) => request<void>('DELETE', `/bingos/${id}`),
  archiveBingo: (id: number) => request<Bingo>('POST', `/bingos/${id}/archive`),
  unarchiveBingo: (id: number) => request<Bingo>('DELETE', `/bingos/${id}/archive`),
  duplicateBingo: (id: number, title: string, shuffle: boolean) =>
    request<Bingo>('POST', `/bingos/${id}/duplicate`, { title, shuffle }),
  validateCell: (id: number, cellId: number, note: string) =>
    request<Bingo>('PUT', `/bingos/${id}/cells/${cellId}/validation`, { note }),
  unvalidateCell: (id: number, cellId: number) =>
    request<Bingo>('DELETE', `/bingos/${id}/cells/${cellId}/validation`),
  bingoHistory: (id: number) => request<BingoEvent[]>('GET', `/bingos/${id}/history`),
  history: () => request<BingoEvent[]>('GET', '/history'),
}
