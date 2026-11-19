const THEME_KEY = 'bingo-theme'

/** Thème mémorisé par l'utilisateur, sinon celui du système. */
export function storedTheme(): 'light' | 'dark' | 'system' {
  try {
    const value = localStorage.getItem(THEME_KEY)
    if (value === 'light' || value === 'dark') return value
  } catch {
    // stockage indisponible (navigation privée) : on suit le système
  }
  return 'system'
}

export function rememberTheme(name: 'light' | 'dark') {
  try {
    localStorage.setItem(THEME_KEY, name)
  } catch {
    // préférence non mémorisée, sans conséquence
  }
}
