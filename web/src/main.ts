import { createApp } from 'vue'
import 'vuetify/styles'
import { createVuetify } from 'vuetify'
import { aliases, mdi } from 'vuetify/iconsets/mdi-svg'
import { fr } from 'vuetify/locale'

import App from './App.vue'
import { router } from './router'
import { storedTheme } from './theme'
import './styles.css'

const vuetify = createVuetify({
  locale: { locale: 'fr', fallback: 'fr', messages: { fr } },
  icons: { defaultSet: 'mdi', aliases, sets: { mdi } },
  theme: {
    defaultTheme: storedTheme(),
    themes: {
      light: {
        dark: false,
        colors: {
          background: '#F6F3EE',
          surface: '#FFFFFF',
          primary: '#5B4BDB',
          secondary: '#F5B700',
          'on-secondary': '#1D1A12',
          success: '#23875B',
          error: '#C93B3B',
          info: '#2F7BD6',
        },
      },
      dark: {
        dark: true,
        colors: {
          background: '#121117',
          surface: '#1D1B26',
          primary: '#A498FF',
          'on-primary': '#16122E',
          secondary: '#FFCB3D',
          'on-secondary': '#1D1A12',
          success: '#4CC38A',
          error: '#F0716B',
          info: '#6AA9F0',
        },
      },
    },
  },
  defaults: {
    VBtn: { rounded: 'lg' },
    VCard: { rounded: 'xl' },
    VTextField: { variant: 'outlined', color: 'primary' },
    VTextarea: { variant: 'outlined', color: 'primary' },
    VDialog: { maxWidth: 480 },
  },
})

createApp(App).use(router).use(vuetify).mount('#app')
