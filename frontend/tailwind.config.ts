import type { Config } from 'tailwindcss'

// Токены-источник истины — DESIGN.md (Kraken-inspired).
// Значения проброшены как CSS-переменные в src/app/styles/tokens.css.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: 'hsl(var(--brand))',
          dark: 'hsl(var(--brand-dark))',
          deep: 'hsl(var(--brand-deep))',
          subtle: 'rgba(133,91,251,0.16)',
        },
        ink: 'hsl(var(--ink))',
        muted: 'hsl(var(--muted))',
        'muted-2': 'hsl(var(--muted-2))',
        border: 'hsl(var(--border))',
        surface: 'hsl(var(--surface))',
        'surface-2': 'hsl(var(--surface-2))',
        success: {
          DEFAULT: 'hsl(var(--success))',
          dark: 'hsl(var(--success-dark))',
        },
        danger: 'hsl(var(--danger))',
      },
      borderRadius: {
        btn: '12px',
        card: '16px',
        badge: '8px',
      },
      fontFamily: {
        display: ['Kraken-Brand', 'IBM Plex Sans', 'Helvetica', 'Arial', 'sans-serif'],
        sans: ['Kraken-Product', 'Helvetica Neue', 'Helvetica', 'Arial', 'sans-serif'],
      },
      boxShadow: {
        subtle: 'rgba(0,0,0,0.03) 0px 4px 24px',
        micro: 'rgba(16,24,40,0.04) 0px 1px 4px',
      },
      screens: {
        xs: '375px',
      },
    },
  },
  plugins: [],
} satisfies Config
