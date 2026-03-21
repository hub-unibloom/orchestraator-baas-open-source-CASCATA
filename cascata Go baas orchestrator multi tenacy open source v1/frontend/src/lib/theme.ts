export type ThemeName = 'cascata-dark' | 'cascata-light' | 'dracula' | 'monokai' | 'hacker';

interface ThemeDefinition {
  name: string;
  colors: {
    surface: {
      base: string;     // Fundo principal do app
      elevated: string; // Modais e painéis flotantes
      glass: string;    // Efeito de vidro com rgba
    };
    content: {
      primary: string;  // Títulos
      muted: string;    // Subtítulos
    };
    accent: {
      primary: string;  // Botões e highlights principais
      emerald: string;  // Sucesso, rotas OK, DB Up
      rose: string;     // Alertas e deleções
      amber: string;    // Draft mode, Environment Switchers
    };
  };
}

// Exemplos de definição estruturada de Temas tipo VSCode
export const themes: Record<ThemeName, ThemeDefinition> = {
  'cascata-dark': {
    name: 'Cascata Space (Default)',
    colors: {
      surface: { base: '10 13 20', elevated: '15 19 26', glass: '15 19 26 / 0.8' },
      content: { primary: '248 250 252', muted: '148 163 184' },
      accent: { primary: '99 102 241', emerald: '16 185 129', rose: '244 63 94', amber: '245 158 11' }
    }
  },
  'dracula': {
    name: 'Dracula Pro',
    colors: {
      surface: { base: '40 42 54', elevated: '68 71 90', glass: '40 42 54 / 0.8' },
      content: { primary: '248 248 242', muted: '98 114 164' },
      accent: { primary: '189 147 249', emerald: '80 250 123', rose: '255 85 85', amber: '241 250 140' }
    }
  },
  'cascata-light': {
    name: 'Cascata Snow',
    colors: {
      surface: { base: '248 250 252', elevated: '255 255 255', glass: '255 255 255 / 0.8' },
      content: { primary: '15 23 42', muted: '71 85 105' },
      accent: { primary: '79 70 229', emerald: '5 150 105', rose: '225 29 72', amber: '217 119 6' }
    }
  },
  'monokai': {
    name: 'Monokai',
    colors: {
      surface: { base: '39 40 34', elevated: '62 61 50', glass: '39 40 34 / 0.8' },
      content: { primary: '248 248 242', muted: '117 113 94' },
      accent: { primary: '249 38 114', emerald: '166 226 46', rose: '249 38 114', amber: '230 219 116' }
    }
  },
  'hacker': {
    name: 'Hacker Terminal',
    colors: {
      surface: { base: '0 0 0', elevated: '5 20 5', glass: '0 0 0 / 0.8' },
      content: { primary: '0 255 0', muted: '0 150 0' },
      accent: { primary: '0 255 0', emerald: '0 255 0', rose: '255 0 0', amber: '200 200 0' }
    }
  }
};

/**
 * Função utilitária para aplicar o tema injetando CSS Variables na raiz.
 * Idealmente chamada dentro de um React.useEffect da <AppProvider /> global.
 */
export function applyTheme(themeName: ThemeName) {
  const root = document.documentElement;
  const theme = themes[themeName] || themes['cascata-dark'];
  
  // Set the logical attribute for CSS selectors
  root.setAttribute('data-theme', themeName);

  // Injetar variáveis CSS de formatação RGB (para o Tailwind com <alpha-value> funcionar bem)
  root.style.setProperty('--color-surface-base', theme.colors.surface.base);
  root.style.setProperty('--color-surface-elevated', theme.colors.surface.elevated);
  root.style.setProperty('--color-surface-glass', theme.colors.surface.glass);
  
  root.style.setProperty('--color-content-primary', theme.colors.content.primary);
  root.style.setProperty('--color-content-muted', theme.colors.content.muted);
  
  root.style.setProperty('--color-accent-primary', theme.colors.accent.primary);
  root.style.setProperty('--color-accent-emerald', theme.colors.accent.emerald);
  root.style.setProperty('--color-accent-rose', theme.colors.accent.rose);
  root.style.setProperty('--color-accent-amber', theme.colors.accent.amber);
}
