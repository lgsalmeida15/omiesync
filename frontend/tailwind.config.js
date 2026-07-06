/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts,js}'],
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Plus Jakarta Sans', 'sans-serif'],
        mono: ['DM Mono', 'monospace']
      },
      colors: {
        accent:  '#00e5ff',
        accent2: '#ff6b35',
        accent3: '#7c3aed',
        // Dark theme
        'dark-bg':      '#080c12',
        'dark-bg2':     '#0d1520',
        'dark-bg3':     '#111d2e',
        'dark-card':    '#0f1c2e',
        'dark-card2':   '#162540',
        'dark-text':    '#e2eaf4',
        'dark-text2':   '#7a90a8',
        'dark-text3':   '#3d5468',
        // Light theme
        'light-bg':     '#f0f4f9',
        'light-card':   '#ffffff',
        'light-text':   '#0f1c2e',
        'light-text2':  '#4a6070',
      },
      borderRadius: {
        card: '12px'
      },
      boxShadow: {
        card: '0 8px 32px rgba(0,0,0,0.45)',
        'card-light': '0 4px 20px rgba(0,0,0,0.1)'
      }
    }
  },
  plugins: []
}
