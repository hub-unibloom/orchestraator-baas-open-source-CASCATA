/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        surface: {
          pit: 'rgb(var(--surface-pit) / <alpha-value>)',
          base: 'rgb(var(--surface-base) / <alpha-value>)',
          raised: 'rgb(var(--surface-raised) / <alpha-value>)',
          elevated: 'rgb(var(--surface-elevated) / <alpha-value>)',
          overlay: 'rgb(var(--surface-overlay) / <alpha-value>)',
        },
        content: {
          primary: 'rgb(var(--content-primary) / <alpha-value>)',
          secondary: 'rgb(var(--content-secondary) / <alpha-value>)',
          muted: 'rgb(var(--content-muted) / <alpha-value>)',
          disabled: 'rgb(var(--content-disabled) / <alpha-value>)',
          inverse: 'rgb(var(--content-inverse) / <alpha-value>)',
        },
        accent: {
          primary: 'rgb(var(--accent-primary) / <alpha-value>)',
          dim: 'rgb(var(--accent-primary-dim) / <alpha-value>)',
          secondary: 'rgb(var(--accent-secondary) / <alpha-value>)',
          danger: 'rgb(var(--accent-danger) / <alpha-value>)',
          warning: 'rgb(var(--accent-warning) / <alpha-value>)',
          success: 'rgb(var(--accent-success) / <alpha-value>)',
          info: 'rgb(var(--accent-info) / <alpha-value>)',
        },
        border: {
          subtle: 'rgb(var(--border-subtle) / 0.05)',
          default: 'rgb(var(--border-default) / 0.09)',
          strong: 'rgb(var(--border-strong) / 0.15)',
          accent: 'rgb(var(--border-accent) / <alpha-value>)',
        }
      },
      fontFamily: {
        sans: ['"Inter Variable"', '"Inter"', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono Variable"', '"Fira Code"', '"Cascadia Code"', 'monospace'],
      },
      animation: {
        'soft-pulse': 'soft-pulse 3s ease-in-out infinite',
      },
      keyframes: {
        'soft-pulse': {
          '0%, 100%': { opacity: '1', transform: 'scale(1)' },
          '50%': { opacity: '0.8', transform: 'scale(1.05)' },
        }
      }
    },
  },
  plugins: [],
}
