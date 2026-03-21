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
} as const;

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
} as const;
