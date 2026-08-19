/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts,js}'],
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      /*
       * Tudo aqui aponta para as variáveis de tokens.css — o Tailwind deixa de
       * ser uma segunda fonte de verdade de cor. O bloco anterior tinha 17 hex
       * fixos (dark-bg, light-card, accent...) e NENHUM deles era usado em
       * lugar algum: era paleta duplicada que só podia divergir com o tempo.
       */
      fontFamily: {
        sans:    ['Inter', 'system-ui', 'sans-serif'],
        display: ['Space Grotesk', 'system-ui', 'sans-serif'],
        // 'mono' mantém o nome por causa das classes font-mono já em uso, mas
        // passa a resolver para a fonte de números do design system.
        mono:    ['Space Grotesk', 'system-ui', 'sans-serif'],
      },
      colors: {
        surface:  'var(--surface)',
        'surface-2': 'var(--surface-2)',
        primary:  'var(--primary)',
        accent:   'var(--accent)',
        success:  'var(--success)',
        warning:  'var(--warning)',
        danger:   'var(--danger)',
        info:     'var(--info)',
        'text-main':   'var(--text)',
        'text-muted':  'var(--text-muted)',
        'text-dim':    'var(--text-dim)',
        'border-token':        'var(--border)',
        'border-token-strong': 'var(--border-strong)',
        // Texto sobre fundo colorido (botão primário). Separado de `surface`
        // porque `text-white` e `bg-white` precisavam de valores diferentes:
        // um é contraste sobre cor, o outro é fundo de cartão.
        oncolor:  'var(--text-oncolor)',
        // Estados de hover. Sem eles o hover: colapsaria na mesma cor da base e
        // o botão deixaria de responder ao ponteiro.
        'primary-hover': 'var(--primary-hover)',
        'warning-fill':  'var(--warning-fill)',
        'danger-fill':   'var(--danger-fill)',
        overlay:  'var(--overlay)',
        'success-weak': 'var(--success-weak)',
        'warning-weak': 'var(--warning-weak)',
        'danger-weak':  'var(--danger-weak)',
        'primary-weak': 'var(--primary-weak)',
      },
      borderRadius: {
        card: 'var(--r)',
      },
      boxShadow: {
        card: 'var(--shadow-md)',
      },
    },
  },
  plugins: [],
}
