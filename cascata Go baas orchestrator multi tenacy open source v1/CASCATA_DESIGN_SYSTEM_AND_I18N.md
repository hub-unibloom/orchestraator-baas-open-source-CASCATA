# Cascata Studio v1.0 — Design System, Layout & i18n

> Este documento é lei. Qualquer componente, página, modal ou drawer criado no Cascata v1 obedece estas regras sem exceção. O objetivo não é apenas consistência — é uma interface que comunica poder, precisão e segurança de nível bancário em cada pixel.

---

## 1. Filosofia Visual

O Cascata não é um SaaS qualquer com tela azul e botão "Get Started". É uma plataforma de infraestrutura para pessoas que entendem o que estão fazendo. A interface deve comunicar isso sem arrogância — elegante, densa de informação, responsiva, e com a sensação de que tudo está vivo e sob controle.

**Referências corretas:** Linear, Vercel Dashboard, Warp Terminal, Raycast.
**Referências incorretas:** Salesforce, qualquer dashboard corporativo de 2018, qualquer coisa que pareça feita no Figma Community em 10 minutos.

**Princípios inegociáveis:**
- **Informação densa, não poluída** — o especialista quer ver tudo; o iniciante não pode se sentir perdido
- **Cada animação tem propósito** — indica transição de estado, confirma ação, guia atenção; nunca decora
- **O backend é visível** — o usuário deve sentir que está tocando algo real, não uma abstração bonita
- **Dark mode é o padrão** — não uma opção, a identidade

---

## 2. Sistema de Tokens (Design Tokens)

Zero cores hardcoded. Zero `bg-slate-900`, `text-indigo-500`, `border-gray-200` direto nos componentes. Tudo referencia tokens semânticos que mudam com o tema.

### 2.1 Superfícies e Profundidade

O sistema tem 5 camadas de profundidade — como um ambiente 3D mínimo:

```css
:root[data-theme="cascata-dark"] {
  /* Superfícies — do mais fundo ao mais alto */
  --surface-pit:      8, 10, 16;    /* #080A10 — fundo de abismos, barra lateral recolhida */
  --surface-base:     11, 14, 21;   /* #0B0E15 — fundo principal do app */
  --surface-raised:   16, 20, 30;   /* #10141E — cards, painéis */
  --surface-elevated: 22, 27, 40;   /* #161B28 — modais, dropdowns */
  --surface-overlay:  30, 36, 54;   /* #1E2436 — tooltips, popovers */

  /* Glass — para elementos sobre outros elementos */
  --surface-glass-bg:     15, 20, 35;   /* com opacity 0.7 + backdrop-blur-xl */
  --surface-glass-border: 255, 255, 255; /* com opacity 0.06 */
}

:root[data-theme="cascata-light"] {
  --surface-pit:      230, 233, 240;
  --surface-base:     242, 244, 248;
  --surface-raised:   252, 253, 255;
  --surface-elevated: 255, 255, 255;
  --surface-overlay:  255, 255, 255;
  --surface-glass-bg:     255, 255, 255;
  --surface-glass-border: 0, 0, 0;
}
```

### 2.2 Conteúdo e Tipografia

```css
:root[data-theme="cascata-dark"] {
  --content-primary:   248, 250, 252;  /* Títulos, dados críticos */
  --content-secondary: 180, 188, 208;  /* Texto normal de interface */
  --content-muted:     100, 112, 140;  /* Labels, placeholders, helpers */
  --content-disabled:  55, 62, 80;     /* Itens desabilitados */
  --content-inverse:   10, 13, 20;     /* Texto sobre fundos claros (badges, etc) */
}
```

### 2.3 Acento e Marca

O acento muda por tema. O componente nunca sabe qual cor é — só sabe que é o acento primário:

```css
:root[data-theme="cascata-dark"] {
  --accent-primary:     99, 102, 241;   /* Indigo — ações principais */
  --accent-primary-dim: 79, 82, 200;    /* Hover state */
  --accent-secondary:   56, 189, 248;   /* Sky — destaques secundários, links */
  --accent-danger:      239, 68, 68;    /* Vermelho — destrutivo */
  --accent-warning:     234, 179, 8;    /* Amarelo — atenção */
  --accent-success:     34, 197, 94;    /* Verde — confirmação */
  --accent-info:        56, 189, 248;   /* Azul claro — informação */
}

:root[data-theme="cascata-aurora"] {
  /* Tema comunitário exemplo — acento completamente diferente */
  --accent-primary:     16, 185, 129;   /* Emerald */
  --accent-secondary:   245, 158, 11;   /* Amber */
}
```

### 2.4 Bordas e Divisores

```css
:root[data-theme="cascata-dark"] {
  --border-subtle:   255, 255, 255;  /* opacity 0.05 — divisores suaves */
  --border-default:  255, 255, 255;  /* opacity 0.09 — bordas de cards */
  --border-strong:   255, 255, 255;  /* opacity 0.15 — bordas de input focus */
  --border-accent:   var(--accent-primary); /* bordas de seleção ativa */
}
```

### 2.5 Mapeamento no Tailwind

```js
// tailwind.config.js
colors: {
  surface: {
    pit:      'rgb(var(--surface-pit) / <alpha-value>)',
    base:     'rgb(var(--surface-base) / <alpha-value>)',
    raised:   'rgb(var(--surface-raised) / <alpha-value>)',
    elevated: 'rgb(var(--surface-elevated) / <alpha-value>)',
    overlay:  'rgb(var(--surface-overlay) / <alpha-value>)',
  },
  content: {
    primary:   'rgb(var(--content-primary) / <alpha-value>)',
    secondary: 'rgb(var(--content-secondary) / <alpha-value>)',
    muted:     'rgb(var(--content-muted) / <alpha-value>)',
    disabled:  'rgb(var(--content-disabled) / <alpha-value>)',
    inverse:   'rgb(var(--content-inverse) / <alpha-value>)',
  },
  accent: {
    primary:   'rgb(var(--accent-primary) / <alpha-value>)',
    dim:       'rgb(var(--accent-primary-dim) / <alpha-value>)',
    secondary: 'rgb(var(--accent-secondary) / <alpha-value>)',
    danger:    'rgb(var(--accent-danger) / <alpha-value>)',
    warning:   'rgb(var(--accent-warning) / <alpha-value>)',
    success:   'rgb(var(--accent-success) / <alpha-value>)',
  },
  border: {
    subtle:  'rgb(var(--border-subtle) / <alpha-value>)',
    default: 'rgb(var(--border-default) / <alpha-value>)',
    strong:  'rgb(var(--border-strong) / <alpha-value>)',
  }
}
```

---

## 3. Tipografia

Fonte única para toda a interface. Sem mistura de fontes decorativas em títulos. O código usa a fonte mono do sistema.

```css
:root {
  /* Interface */
  --font-sans: 'Inter Variable', 'Inter', system-ui, sans-serif;
  /* Código, SQL, logs, IDs */
  --font-mono: 'JetBrains Mono Variable', 'Fira Code', 'Cascadia Code', monospace;
}
```

**Escala tipográfica via tokens:**
```css
:root {
  --text-xs:   0.70rem;  /* 11px — labels, badges, timestamps */
  --text-sm:   0.8125rem; /* 13px — texto secundário, helpers */
  --text-base: 0.875rem; /* 14px — texto principal da interface */
  --text-md:   1rem;     /* 16px — títulos de seção */
  --text-lg:   1.125rem; /* 18px — títulos de página */
  --text-xl:   1.25rem;  /* 20px — títulos de modal */
  --text-2xl:  1.5rem;   /* 24px — títulos maiores */
}
```

**Regras:**
- Toda string de dados (UUID, slug, chaves, SQL) usa `font-mono`
- Números de métricas e contadores usam `font-mono` com `tabular-nums`
- Zero `font-bold` em textos longos — peso máximo para corpo de texto é `font-medium`

---

## 4. Sistema de Motion (Linguagem de Animação)

Toda animação segue um vocabulário consistente. Nenhum componente inventa sua própria duração ou easing.

```ts
// src/lib/motion.ts — importado por todos os componentes que animam
export const MOTION = {
  // Micro-interações — feedback imediato
  micro: { duration: 0.1, ease: [0.4, 0, 0.2, 1] },

  // Transições de estado — aparecimento de elementos
  swift: { duration: 0.18, ease: [0.4, 0, 0.2, 1] },

  // Entrada de painéis, modais, drawers
  smooth: { duration: 0.28, ease: [0.4, 0, 0.2, 1] },

  // Transições de página, mudanças grandes de layout
  flow:   { duration: 0.4,  ease: [0.4, 0, 0.2, 1] },

  // Spring — elementos que "respondem" ao toque (drag, resize)
  spring: { type: 'spring', stiffness: 400, damping: 35 },
} as const

// Variantes padrão para elementos que entram na tela
export const VARIANTS = {
  fadeUp: {
    hidden:  { opacity: 0, y: 6 },
    visible: { opacity: 1, y: 0, transition: MOTION.smooth },
  },
  fadeIn: {
    hidden:  { opacity: 0 },
    visible: { opacity: 1, transition: MOTION.swift },
  },
  scaleIn: {
    hidden:  { opacity: 0, scale: 0.97 },
    visible: { opacity: 1, scale: 1, transition: MOTION.smooth },
  },
  slideRight: {
    hidden:  { opacity: 0, x: -8 },
    visible: { opacity: 1, x: 0, transition: MOTION.smooth },
  },
} as const
```

**O que anima e o que não anima:**
- ✅ Entrada de modais, drawers, tooltips, dropdowns
- ✅ Feedback de hover em botões e itens de lista
- ✅ Transições de estado (loading → dados, vazio → preenchido)
- ✅ Drag & drop com spring physics
- ✅ Número contadores quando o valor muda (SSE update)
- ❌ Animações de scroll decorativas
- ❌ Loop animations em elementos não-interativos
- ❌ Qualquer coisa que atrase percepção de resposta > 150ms

---

## 5. Glassmorphism — Como Fazer Certo

Glass não é só `backdrop-blur`. É uma composição de 4 elementos simultâneos:

```css
.glass-panel {
  background: rgb(var(--surface-glass-bg) / 0.72);
  backdrop-filter: blur(24px) saturate(1.4);
  -webkit-backdrop-filter: blur(24px) saturate(1.4);
  border: 1px solid rgb(var(--surface-glass-border) / 0.07);
  box-shadow:
    0 1px 0 0 rgb(255 255 255 / 0.04) inset,  /* brilho no topo */
    0 0 0 1px rgb(0 0 0 / 0.12),              /* borda sutil externa */
    0 8px 32px -8px rgb(0 0 0 / 0.4);         /* sombra de elevação */
}
```

**Onde usar glass:**
- Modais sobre conteúdo vivo (SSE ativo por baixo)
- Command Palette (Ctrl+K)
- Floating toolbars no DB Explorer
- Notificações e toasts

**Onde NÃO usar glass:**
- Sidebar principal — é estrutura, não flutua
- Cards de dados — legibilidade acima de tudo
- Tabelas — zero blur sobre texto denso

---

## 6. Layout Adaptativo e Persistência

### 6.1 Estrutura de Layout

```
AppShell
├── Sidebar (redimensionável, reordenável, colapsável)
├── ContentArea
│   ├── TopBar (breadcrumb, ações globais, busca)
│   ├── PrimaryPanel (conteúdo principal)
│   └── SecondaryPanel (split view — opcional, redimensionável)
└── BottomBar (status do cluster, SSE indicator, shortcuts)
```

### 6.2 O que persiste no banco

Tudo que o usuário configurou sobre como trabalha é salvo em `/api/control/users/preferences` via PATCH com debounce de 2s:

```ts
interface LayoutPreferences {
  sidebarWidth: number           // px — padrão 240
  sidebarOrder: string[]         // IDs dos módulos na ordem do usuário
  sidebarCollapsed: boolean
  bottomBarHeight: number        // px — terminal/logs embutido
  splitViewEnabled: boolean
  splitViewRatio: number         // 0.0 a 1.0
  dbExplorerPinnedTables: string[] // slugs de tabelas fixadas
  activeTheme: string            // ID do tema ativo
  activeLocale: string           // código do idioma
  densityMode: 'comfortable' | 'compact' | 'spacious'
}
```

### 6.3 Density Mode

Três modos de densidade que ajustam padding, espaçamento e tamanho de fonte via tokens:

```css
:root[data-density="compact"] {
  --spacing-row: 0.375rem;   /* linhas de tabela compactas */
  --spacing-item: 0.5rem;    /* itens de lista */
  --spacing-section: 1rem;   /* espaço entre seções */
}
:root[data-density="comfortable"] { /* padrão */
  --spacing-row: 0.625rem;
  --spacing-item: 0.75rem;
  --spacing-section: 1.5rem;
}
:root[data-density="spacious"] {
  --spacing-row: 0.875rem;
  --spacing-item: 1rem;
  --spacing-section: 2rem;
}
```

### 6.4 Hook de Layout

```ts
// src/hooks/useLayoutStore.ts
// Zustand store — único ponto de verdade para preferências de layout
const useLayoutStore = create<LayoutStore>()(
  persist(
    (set, get) => ({
      preferences: defaultPreferences,
      updatePreference: (key, value) => {
        set(state => ({
          preferences: { ...state.preferences, [key]: value }
        }))
        // Debounce PATCH para API
        debouncedSync(get().preferences)
      },
    }),
    { name: 'cascata-layout' } // localStorage key
  )
)
```

---

## 7. Internacionalização (i18n)

### 7.1 A Regra

**Zero string literal visível ao usuário no JSX.** Sem exceção.

```tsx
// ❌ Proibido
<button>Delete Table</button>
<p>Are you sure you want to delete {tableName}?</p>

// ✅ Correto
<button>{t('database.actions.delete_table')}</button>
<p>{t('database.confirm.delete_table', { name: tableName })}</p>
```

### 7.2 Estrutura de Namespaces

Carregamento lazy por namespace — não carrega 5MB de uma vez:

```
frontend/src/locales/
├── en/                    (base — sempre carregado)
│   ├── common.json        (Cancel, Save, Delete, Confirm, Error...)
│   ├── auth.json          (Login, Logout, OTP, MFA...)
│   ├── database.json      (Tables, Columns, RLS, Indexes...)
│   ├── automation.json    (Triggers, Nodes, Flows, Webhooks...)
│   ├── storage.json       (Upload, Download, Quota, Providers...)
│   ├── security.json      (Vault, Keys, Panic Mode, Masking...)
│   ├── settings.json      (Preferences, Theme, Members, Agents...)
│   └── errors.json        (Todos os códigos de erro do backend)
├── pt-BR/
│   └── (mesma estrutura — carregado dinamicamente)
├── es/
├── zh/
└── fr/
```

### 7.3 Tratamento de Plurais e Contexto

```ts
// Plurais
t('database.row_count', { count: rows.length })
// en: "1 row" / "42 rows"

// Contexto de gênero (para idiomas que precisam)
t('member.role', { role: 'admin', context: 'female' })

// Interpolação rica (componentes dentro de traduções)
t('security.panic_warning', {
  tenant: <strong>{tenantSlug}</strong>,
  action: <code>LOCK</code>
})
```

### 7.4 Códigos de Erro do Backend como Chaves i18n

Todo código semântico retornado pela API (`AUTH_FAILED`, `QUOTA_EXCEEDED`, `RLS_VIOLATION`) tem entrada no `errors.json`:

```json
{
  "AUTH_FAILED": "Authentication failed. Check your credentials.",
  "QUOTA_EXCEEDED": "Storage quota exceeded for this tenant.",
  "RLS_VIOLATION": "Access denied by Row Level Security policy.",
  "TENANT_NOT_FOUND": "Project not found or you don't have access."
}
```

O frontend nunca exibe o código cru para o usuário — sempre traduzido.

---

## 8. Componentes de Dados — Padrões Específicos

### 8.1 Números e Métricas

```tsx
// Sempre font-mono + tabular-nums para números que mudam
<span className="font-mono tabular-nums text-content-primary">
  {formatMetric(connectionCount)}
</span>

// Formatação consistente
formatMetric(1234567)  // → "1.2M"
formatMetric(1234)     // → "1,234"
formatBytes(1073741824) // → "1.0 GB"
formatLatency(0.0034)   // → "3.4ms"
```

### 8.2 Status Indicators (SSE Live)

Todo indicador de dado ao vivo tem um pulso sutil que para quando o SSE disconnecta:

```tsx
// Dot pulsante — verde quando conectado, cinza quando offline
<LiveIndicator connected={sseConnected} />
// Internamente: animação CSS de 2s loop, para com `animation: none` quando offline
```

### 8.3 Tabelas de Dados

```tsx
// Virtualização obrigatória para > 100 linhas
// TanStack Virtual ou react-window
// Nunca renderiza linha não visível
<VirtualTable
  data={rows}
  columns={columns}
  estimateSize={() => 32} // px por linha no density atual
  overscan={5}
/>
```

---

## 9. Temas da Comunidade

Temas são arquivos JSON hospedados externamente, carregados em runtime — o bundle nunca cresce:

```ts
interface CommunityTheme {
  id: string
  name: string
  author: string
  version: string
  tokens: Partial<ThemeTokens> // sobrescreve apenas o que muda
}

// Carregamento
const theme = await fetch(`https://themes.cascata.dev/${themeId}.json`)
applyTheme(theme) // injeta tokens no :root via JS
```

**Temas nativos inclusos:**
- `cascata-dark` — Deep Space (padrão, azul com branco gelo celeste)
- `cascata-light` — Snow
- `cascata-midnight` — Preto absoluto para OLED
- `cascata-aurora` — Verde esmeralda (alternativa vibrante)

---

## 10. Responsividade

Três breakpoints reais — não os genéricos do Tailwind:

```ts
const BREAKPOINTS = {
  mobile:  640,  // < 640px — layout colapsado, sidebar em drawer
  tablet:  1024, // 640–1024px — sidebar compacta, split view desabilitado
  desktop: 1280, // > 1024px — layout completo
} as const
```

**Regra:** nenhuma feature é removida em mobile — é adaptada. Panic Mode, Vault, tudo acessível. A complexidade se adapta, não desaparece.

---

## 11. Accessibility (a11y)

Mínimos obrigatórios — não opcional:
- Todo elemento interativo tem `aria-label` descritivo via i18n
- Contraste mínimo 4.5:1 para texto normal, 3:1 para texto grande
- `prefers-reduced-motion`: todas as animações Framer Motion respeitam `useReducedMotion()`
- Navegação por teclado completa em modais, dropdowns e command palette
- `role` e `aria-*` corretos em componentes customizados (não usar `div` clicável sem `role="button"`)

---

## Conclusão

Qualquer componente criado no Cascata v1 obedece:
1. **Tokens semânticos** — zero cor hardcoded, zero string hardcoded
2. **Motion vocabulary** — duração e easing do `MOTION` object, nunca inventados
3. **Layout store** — qualquer mudança de layout persiste via `useLayoutStore`
4. **i18n** — `t('namespace.key')` em todo texto visível ao usuário, incluindo erros
5. **Responsividade real** — adaptação por breakpoint, nenhuma feature removida
6. **a11y** — aria e contraste verificados antes de considerar o componente pronto