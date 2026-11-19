const THEME_KEY = 'bingo-theme'

/** The theme the user picked, or the system's if they never chose. */
export function storedTheme(): 'light' | 'dark' | 'system' {
  try {
    const value = localStorage.getItem(THEME_KEY)
    if (value === 'light' || value === 'dark') return value
  } catch {
    // storage unavailable (private browsing): follow the system
  }
  return 'system'
}

export function rememberTheme(name: 'light' | 'dark') {
  try {
    localStorage.setItem(THEME_KEY, name)
  } catch {
    // preference not persisted, which is harmless
  }
}
