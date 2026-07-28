import 'vuetify/styles'
import '@/styles/theme.scss'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'

const theme = {
  light: {
    dark: false,
    colors: {
      primary: '#1976D2',
      'primary-darken-1': '#1565C0',
      secondary: '#424242',
      'secondary-darken-1': '#303030',
      accent: '#2196F3',
      success: '#4CAF50',
      warning: '#FF9800',
      error: '#F44336',
      info: '#2196F3',
    },
  },
  dark: {
    dark: true,
    colors: {
      primary: '#42A5F5',
      'primary-darken-1': '#1E88E5',
      secondary: '#757575',
      'secondary-darken-1': '#616161',
      accent: '#64B5F6',
      success: '#66BB6A',
      warning: '#FFA726',
      error: '#EF5350',
      info: '#42A5F5',
    },
  },
}

const THEME_KEY = 'talkiepi:theme'
const savedTheme = localStorage.getItem(THEME_KEY)
const initialTheme = savedTheme === 'dark' || savedTheme === 'light' ? savedTheme : 'dark'

export default createVuetify({
  components,
  directives,
  theme: {
    defaultTheme: initialTheme,
    themes: theme,
  },
})

export { THEME_KEY }
