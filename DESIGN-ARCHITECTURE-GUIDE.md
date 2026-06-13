# WhatsTracker 2.0 - Guia Completo de Design

> **Documento de Referência de Design e UI/UX**  
> Este guia documenta todo o sistema de design, padrões visuais, componentes e estrutura de interface do WhatsTracker 2.0 para ser usado como referência em outros projetos.

---

## 📚 Índice

1. [Filosofia de Design](#filosofia-de-design)
2. [Sistema de Cores e Tokens](#sistema-de-cores-e-tokens)
3. [Tipografia](#tipografia)
4. [Componentes Base](#componentes-base)
5. [Layouts e Navegação](#layouts-e-navegação)
6. [Sistema de Temas (Dark/Light)](#sistema-de-temas)
7. [Responsividade e Mobile](#responsividade-e-mobile)
8. [Iconografia e Assets](#iconografia-e-assets)
9. [Padrões de Interação](#padrões-de-interação)
10. [Estados Visuais](#estados-visuais)

---

## Filosofia de Design

O WhatsTracker 2.0 segue princípios de design modernos e profissionais:

### Princípios Core

1. **Design Tokens First** - Todas as cores, espaçamentos e estilos vêm de variáveis CSS reutilizáveis
2. **Dark Mode Nativo** - Desenhado primeiro para tema escuro, com suporte completo a tema claro
3. **Mobile-First** - Interface pensada para dispositivos móveis desde o início
4. **Componentes Atômicos** - Sistema baseado em componentes pequenos e reutilizáveis
5. **Acessibilidade** - Alto contraste, navegação por teclado, semântica HTML
6. **Performance Visual** - Animações suaves (60fps), lazy loading, skeleton states

### Stack Visual
- **Tailwind CSS 3.4** - Utility-first framework
- **CSS Custom Properties** - Tokens de tema dinâmicos
- **Google Fonts** - Instrument Sans (títulos) + Inter (corpo)
- **SVG Icons** - Iconografia inline customizada
- **Chart.js** - Gráficos e visualizações de dados



---

## Sistema de Cores e Tokens

### Arquitetura de Cores

O sistema usa **CSS Custom Properties (variáveis CSS)** para permitir troca de tema dinâmica sem recarregar a página.

#### Arquivo: `app/assets/css/main.css`

```css
/* TEMA ESCURO (padrão) */
:root {
  /* Superfícies - do fundo ao mais elevado */
  --surface-1: #2a2a2a;            /* Fundo da página */
  --surface-2: #1f1f1f;            /* Cards, sidebar, modais */
  --surface-3: #2a2a2a;            /* Hover, dropdowns */
  --surface-deep: #111111;         /* Inputs dentro de modais */
  --surface-soft: rgba(255, 255, 255, 0.05);
  --surface-soft-2: rgba(255, 255, 255, 0.03);

  /* Bordas */
  --border: rgba(255, 255, 255, 0.1);
  --border-soft: rgba(255, 255, 255, 0.08);
  --border-faint: rgba(255, 255, 255, 0.05);

  /* Texto */
  --text-strong: #f5f5f5;          /* Títulos principais */
  --text-1: #e5e5e5;               /* Texto primário */
  --text-2: #a3a3a3;               /* Texto secundário */
  --text-3: #737373;               /* Texto terciário/hint */

  /* Marca (constante entre temas) */
  --brand: #1C87C1;                /* Azul primário */
  --brand-strong: #1570a3;         /* Hover */
  --brand-soft: rgba(28, 135, 193, 0.1);
  --brand-soft-2: rgba(28, 135, 193, 0.14);
  --brand-border: rgba(28, 135, 193, 0.3);

  /* Estados */
  --success: #22c55e;
  --success-soft: rgba(34, 197, 94, 0.1);
  --danger: #ef4444;
  --danger-soft: rgba(239, 68, 68, 0.1);
  --warning: #f59e0b;
  --warning-soft: rgba(245, 158, 11, 0.1);

  /* Sombras */
  --shadow-card: 0 2px 12px rgba(0, 0, 0, 0.18);
  --shadow-elevated: 0 8px 32px rgba(0, 0, 0, 0.5);

  color-scheme: dark;
}

/* TEMA CLARO */
:root[data-theme='light'] {
  --surface-1: #f4f5f7;
  --surface-2: #ffffff;
  --surface-3: #f9fafb;
  --surface-deep: #f3f4f6;
  --surface-soft: rgba(0, 0, 0, 0.04);
  --surface-soft-2: rgba(0, 0, 0, 0.02);

  --border: rgba(15, 23, 42, 0.12);
  --border-soft: rgba(15, 23, 42, 0.08);
  --border-faint: rgba(15, 23, 42, 0.05);

  --text-strong: #0f172a;
  --text-1: #1f2937;
  --text-2: #4b5563;
  --text-3: #6b7280;

  --brand-soft: rgba(28, 135, 193, 0.08);
  --brand-soft-2: rgba(28, 135, 193, 0.12);
  --brand-border: rgba(28, 135, 193, 0.35);

  --success-soft: rgba(34, 197, 94, 0.12);
  --danger-soft: rgba(239, 68, 68, 0.1);
  --warning-soft: rgba(245, 158, 11, 0.12);

  --shadow-card: 0 1px 3px rgba(15, 23, 42, 0.08), 0 1px 2px rgba(15, 23, 42, 0.04);
  --shadow-elevated: 0 10px 32px rgba(15, 23, 42, 0.12);

  color-scheme: light;
}
```

### Paleta de Cores Tailwind

Além dos tokens CSS, o Tailwind está configurado com cores específicas:

```typescript
// tailwind.config.ts
colors: {
  primary: '#1C87C1',      // Azul principal
  secondary: '#10B981',    // Verde
  danger: '#EF4444',       // Vermelho
  warning: '#F59E0B',      // Laranja
  info: '#06B6D4',         // Ciano
  
  // Tema escuro
  dark: {
    bg: '#333333',         // Fundo principal
    surface: '#404040',    // Cards/surfaces
    border: '#4a4a4a',     // Bordas
    text: '#e5e5e5',       // Texto principal
    muted: '#a3a3a3'       // Texto secundário
  }
}
```


### Uso dos Tokens

**✅ Correto** - Usar tokens CSS:
```css
.card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  color: var(--text-1);
}
```

**❌ Evitar** - Hardcoded ou classes Tailwind `dark:`:
```css
.card {
  background: #1f1f1f;  /* ❌ não adapta ao tema */
}
```

---

## Tipografia

### Fontes

```css
@import url('https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@300;400;500;600;700&display=swap');
```

**Instrument Sans** - Títulos e elementos de destaque
- Pesos: 300, 400, 500, 600, 700
- Caracteres: Latin extended

**Inter** (fallback padrão do Tailwind) - Corpo de texto
- Sistema completo de pesos
- Melhor legibilidade em tamanhos pequenos

### Hierarquia Tipográfica

```css
h1 { font-size: 3xl (1.875rem); font-weight: 700; }
h2 { font-size: 2xl (1.5rem); font-weight: 600; }
h3 { font-size: xl (1.25rem); font-weight: 600; }

/* Corpo */
body { font-size: base (1rem); font-weight: 400; }

/* Secundário */
.text-sm { font-size: 0.875rem; }
.text-xs { font-size: 0.75rem; }
```

### Classes Tailwind Customizadas

```typescript
// tailwind.config.ts
fontFamily: {
  sans: ['Inter', ...defaultTheme.fontFamily.sans],
  heading: ['Instrument Sans', 'Inter', ...defaultTheme.fontFamily.sans]
}
```

**Uso:**
```html
<h1 class="font-heading text-3xl font-bold">Título Principal</h1>
<p class="font-sans text-base">Texto do corpo</p>
```


---

## Componentes Base

### Arquitetura de Componentes

Todos os componentes base estão em `app/components/` e seguem o padrão `Base*.vue`:

```
app/components/
├── BaseButton.vue       # Botões com variantes
├── BaseInput.vue        # Inputs de formulário
├── BaseDropdown.vue     # Seletor dropdown
├── BaseAvatar.vue       # Avatar de usuário
├── BaseToast.vue        # Notificações toast
├── BaseUpload.vue       # Upload de arquivos
├── BaseForm.vue         # Wrapper de formulários
└── ConfirmModal.vue     # Modal de confirmação
```

### BaseButton

**Variantes:**
- `primary` - Ação principal (azul sólido)
- `secondary` - Ação secundária (verde)
- `danger` - Ação destrutiva (vermelho)
- `ghost` - Botão transparente
- `outline` - Com borda

**Tamanhos:**
- `sm` - Pequeno (px-3 py-1.5)
- `md` - Médio (px-4 py-2) [padrão]
- `lg` - Grande (px-5 py-2.5)

**Exemplo:**
```vue
<BaseButton 
  variant="primary" 
  size="md" 
  :loading="isSubmitting"
  @click="handleSubmit"
>
  <template #iconLeft>
    <svg>...</svg>
  </template>
  Salvar alterações
</BaseButton>
```

**Estados visuais:**
- Hover: Escurece levemente a cor
- Active: Escurece mais
- Disabled: opacity 50%
- Loading: Spinner animado centralizado


### BaseInput

**Features:**
- Label com indicador de required (*)
- Ícone à esquerda (slot)
- Estados: error, disabled, focus
- Mensagem de erro e hint

**Exemplo:**
```vue
<BaseInput
  v-model="email"
  type="email"
  label="E-mail"
  placeholder="seu@email.com"
  :error="errors.email"
  hint="Usaremos apenas para login"
  required
>
  <template #icon>
    <svg>...</svg> <!-- Ícone de email -->
  </template>
</BaseInput>
```

**Estilos:**
```css
/* Padrão */
padding: 0.55rem 0.875rem;
border-radius: 8px;
border: 1px solid var(--border);
background: var(--surface-2);

/* Focus */
border-color: var(--brand);
box-shadow: 0 0 0 3px var(--brand-soft);

/* Error */
border-color: var(--danger);
box-shadow: 0 0 0 3px var(--danger-soft);
```

### BaseDropdown

Seletor customizado com busca integrada:

**Props:**
- `options` - Array de { value, label }
- `searchable` - Habilita busca
- `placeholder`
- `emptyText` - Texto quando vazio

**Exemplo:**
```vue
<BaseDropdown
  v-model="selectedClient"
  :options="clientOptions"
  placeholder="Selecione um cliente"
  searchable
  search-placeholder="Buscar..."
  empty-text="Nenhum cliente encontrado"
/>
```


### BaseAvatar

Avatar de usuário com fallback para iniciais:

**Tamanhos:**
- `sm` - 32px
- `md` - 40px [padrão]
- `lg` - 48px
- `xl` - 64px

**Lógica:**
1. Se `src` existir → mostra imagem
2. Se não → gera iniciais do `name`
3. Cor de fundo aleatória baseada no hash do nome

**Exemplo:**
```vue
<BaseAvatar
  :name="user.name"
  :src="user.avatarUrl"
  size="md"
/>
```

### BaseToast

Sistema de notificações toast (canto superior direito):

**Tipos:**
- `success` - Verde com ícone de check
- `error` - Vermelho com ícone de X
- `warning` - Laranja com ícone de alerta
- `info` - Azul com ícone de info

**Uso via composable:**
```typescript
const toast = useToast()

toast.success('Título', 'Mensagem de sucesso')
toast.error('Erro', 'Algo deu errado')
toast.warning('Atenção', 'Cuidado com isso')
toast.info('Info', 'Apenas informando')
```

**Comportamento:**
- Auto-dismiss após 5 segundos
- Empilhamento vertical
- Animação de entrada/saída suave
- Máximo 3 toasts simultâneos


---

## Layouts e Navegação

### Estrutura de Layouts

**2 layouts principais:**

#### 1. DefaultLayout
- Para páginas públicas (login, cadastro, esqueci senha)
- Header simples + footer
- Sem sidebar
- Centro da tela, max-width 7xl

#### 2. AppLayout
- Para área autenticada
- Sidebar fixa à esquerda (desktop)
- Bottom nav (mobile/tablet)
- Background gradiente sutil
- Container fluido com padding

```
┌─────────────────────────────────┐
│  AppBackground (gradiente)      │
│  ┌───────┬──────────────────┐   │
│  │       │                  │   │
│  │ Side  │   Main Content   │   │
│  │ bar   │                  │   │
│  │       │                  │   │
│  └───────┴──────────────────┘   │
│            │                     │
│      BottomNav (mobile)          │
└─────────────────────────────────┘
```

### Sidebar (Desktop)

**Dimensões:**
- Largura: 280px
- Altura: 100vh
- Posição: fixed left
- Z-index: 50

**Estrutura:**
```
┌────────────────────┐
│  Logo              │  ← Logo da empresa
├────────────────────┤
│                    │
│  Navigation        │  ← Links principais
│  • Dashboard       │
│  • Clientes        │
│  • APIs            │
│  • Configurações   │
│                    │
├────────────────────┤
│  Version Badge     │  ← Tipo de conta
│  User Info         │  ← Avatar + nome
│  Logout            │
└────────────────────┘
```


**Componentes:**

```vue
<!-- SidebarLink.vue -->
<NuxtLink class="sidebar-link">
  <span class="icon">...</span>
  <span class="label">Dashboard</span>
  <span class="badge">PRO</span> <!-- opcional -->
</NuxtLink>
```

**Estados:**
- Normal: texto muted, ícone muted
- Hover: background soft, texto mais forte
- Active: background brand-soft, texto brand, borda esquerda brand

**SidebarGroup** (collapsible):
```vue
<SidebarGroup 
  label="Configurações" 
  icon="settings"
  base-path="/configuracoes"
>
  <SidebarLink to="/configuracoes" label="Geral" />
  <SidebarLink to="/configuracoes/integracoes" label="Integrações" />
</SidebarGroup>
```

### MobileBottomNav

**Características:**
- Aparece apenas em `max-width: 1023px`
- Fixed bottom com safe-area para iOS
- Scroll horizontal se muitos itens
- Ícones 20px + label pequeno

**Layout:**
```
┌──────────────────────────────────┐
│ [🏠] [👥] [⚙️] [👤] [📊]         │
│ Home Cli  Conf Perf  API         │
└──────────────────────────────────┘
  ↑ padding-bottom: safe-area
```

**Estilos:**
```css
position: fixed;
bottom: 0;
background: var(--surface-2);
box-shadow: 0 -4px 16px rgba(0, 0, 0, 0.25);
padding-bottom: env(safe-area-inset-bottom);
z-index: 100;
```


### AppBackground

Componente decorativo de fundo:

```vue
<!-- AppBackground.vue -->
<div class="app-background">
  <!-- Gradiente sutil radial do topo -->
</div>
```

```css
.app-background {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 60vh;
  background: radial-gradient(
    ellipse at top,
    rgba(28, 135, 193, 0.08) 0%,
    transparent 60%
  );
  pointer-events: none;
  z-index: 0;
}
```

---

## Sistema de Temas

### Implementação

O tema é controlado pelo atributo `data-theme` no `<html>`:

```html
<!-- Tema escuro (padrão) -->
<html data-theme="dark">

<!-- Tema claro -->
<html data-theme="light">
```

### Composable useTheme

```typescript
const { theme, isDark, isLight, toggle, setTheme } = useTheme()

// Alternar tema
toggle()

// Definir específico
setTheme('light')  // ou 'dark'

// Verificar tema atual
if (isDark.value) { ... }
```

### Persistência

**3 camadas:**
1. **Cookie local** - Sobrevive refresh
2. **State global** - Reativo no Vue
3. **Banco de dados** - `users.theme` (quando logado)

### Toggle de Tema (UI)

Botão comum em configurações:

```vue
<button @click="toggle" class="theme-toggle">
  <svg v-if="isDark">🌙 Tema Escuro</svg>
  <svg v-else>☀️ Tema Claro</svg>
</button>
```


---

## Responsividade e Mobile

### Breakpoints Tailwind

```typescript
screens: {
  'sm': '640px',    // Mobile grande / Tablet pequeno
  'md': '768px',    // Tablet
  'lg': '1024px',   // Desktop pequeno
  'xl': '1280px',   // Desktop
  '2xl': '1536px'   // Desktop grande
}
```

### Estratégia Mobile-First

**Regra:** Escrever CSS para mobile primeiro, adicionar breakpoints para desktop

**❌ Evitar:**
```css
.container {
  width: 1200px;  /* desktop */
}

@media (max-width: 768px) {
  .container {
    width: 100%;  /* mobile */
  }
}
```

**✅ Correto:**
```css
.container {
  width: 100%;  /* mobile */
}

@media (min-width: 768px) {
  .container {
    max-width: 1200px;  /* desktop */
  }
}
```

### Padrões Responsivos Comuns

#### Layout Principal

```css
/* Mobile: sem sidebar, usa bottom nav */
@media (max-width: 1023px) {
  .sidebar { display: none; }
  .app-main { 
    margin-left: 0; 
    padding-bottom: calc(95px + env(safe-area-inset-bottom));
  }
  .mobile-bottom-nav { display: block; }
}

/* Desktop: sidebar fixa, sem bottom nav */
@media (min-width: 1024px) {
  .sidebar { display: flex; }
  .app-main { margin-left: 280px; }
  .mobile-bottom-nav { display: none; }
}
```


#### Header de Página

```vue
<!-- Desktop: horizontal -->
<div class="page-header">
  <div class="header-left">...</div>
  <div class="header-actions">...</div>
</div>

<!-- Mobile: vertical stack -->
@media (max-width: 640px) {
  .page-header {
    flex-direction: column;
    gap: 0.75rem;
  }
  
  .header-actions {
    width: 100%;
    flex-wrap: wrap;
  }
}
```

#### Cards/Grid

```html
<!-- Responsivo com Tailwind -->
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
  <Card />
  <Card />
  <Card />
</div>
```

### Safe Area (iOS)

Para notches, ilhas dinâmicas e botão home:

```css
/* Top padding para não ficar embaixo do notch */
padding-top: calc(2rem + env(safe-area-inset-top, 0));

/* Bottom padding para não ficar embaixo do indicador */
padding-bottom: calc(95px + env(safe-area-inset-bottom, 0));
```

### Touch Targets

**Mínimo recomendado:** 44x44px (Apple HIG)

Todos os botões e links interativos respeitam:
```css
min-height: 44px;
min-width: 44px;
```


---

## Iconografia e Assets

### Sistema de Ícones

**Abordagem:** SVG inline (não font icons)

**Vantagens:**
- Tamanho customizável via CSS
- Colorização via `currentColor`
- Sem dependência externa
- Melhor performance

**Padrão:**
```vue
<svg 
  xmlns="http://www.w3.org/2000/svg" 
  class="h-5 w-5" 
  fill="none" 
  viewBox="0 0 24 24" 
  stroke="currentColor"
  stroke-width="2"
>
  <path stroke-linecap="round" stroke-linejoin="round" d="..." />
</svg>
```

### Fontes de Ícones

1. **Heroicons** (majoritários) - linha fina, moderno
2. **Ícones customizados** - WhatsApp, Meta, etc

### Logos

**Localização:** `/public/img/`

```
/public/img/
├── logo-transparente.png           # Dark theme
├── logo-transparente-claro.png     # Light theme
├── favicon.ico
└── icons/
    ├── icon-32.png
    ├── icon-180.png   # Apple touch
    ├── icon-192.png   # PWA
    └── icon-512.png   # PWA
```

**Uso com tema:**
```vue
<img 
  :src="isLight ? '/img/logo-claro.png' : '/img/logo-escuro.png'" 
  alt="Logo"
/>
```


---

## Padrões de Interação

### Animações e Transições

**Princípio:** Suave mas perceptível (200-300ms)

#### Transições Padrão

```css
/* Hover states */
transition: all 0.2s ease;

/* Backgrounds e cores */
transition: background 0.15s, border-color 0.15s;

/* Transformações */
transition: transform 0.18s;

/* Opacidade */
transition: opacity 0.2s ease;
```

#### Animações Específicas

**Spinner de loading:**
```css
@keyframes spin {
  to { transform: rotate(360deg); }
}

.spinner {
  animation: spin 0.8s linear infinite;
}
```

**Skeleton pulse:**
```css
@keyframes skeleton-pulse {
  0%, 100% { opacity: 0.5; }
  50% { opacity: 0.8; }
}

.skeleton {
  animation: skeleton-pulse 1.5s ease-in-out infinite;
}
```

**Toast slide-in:**
```css
@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}
```


### Hover States

**Padrão para botões:**
```css
.button {
  transition: all 0.2s ease;
}

.button:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.button:active {
  transform: translateY(0);
}
```

**Cards interativos:**
```css
.card {
  transition: box-shadow 0.2s, transform 0.2s;
}

.card:hover {
  box-shadow: var(--shadow-elevated);
  transform: translateY(-2px);
}
```

### Focus States

**Acessibilidade via teclado:**
```css
.button:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.input:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}
```

### Loading States

**3 abordagens:**

1. **Spinner no botão**
```vue
<BaseButton :loading="isSubmitting">
  Salvar
</BaseButton>
```

2. **Skeleton screens**
```vue
<div v-if="loading" class="skeleton">
  <div class="skeleton-line"></div>
  <div class="skeleton-line short"></div>
</div>
<div v-else>
  {{ content }}
</div>
```

3. **Loading overlay**
```vue
<div class="relative">
  <div v-if="loading" class="loading-overlay">
    <div class="spinner"></div>
  </div>
  <Content />
</div>
```


---

## Estados Visuais

### Empty States

Quando não há dados para exibir:

```vue
<div class="empty-state">
  <svg class="empty-icon">...</svg>
  <h3 class="empty-title">Nenhum cliente cadastrado</h3>
  <p class="empty-description">
    Configure sua integração e adicione clientes para começar.
  </p>
  <BaseButton variant="primary" @click="onAction">
    Adicionar Cliente
  </BaseButton>
</div>
```

**Estilos:**
```css
.empty-state {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 4rem 2rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.empty-icon {
  width: 64px;
  height: 64px;
  color: var(--text-2);
}
```

### Error States

**Inline errors (formulários):**
```vue
<BaseInput
  v-model="email"
  :error="errors.email"
/>
```

**Page-level errors:**
```vue
<div class="error-banner">
  <svg class="error-icon">❌</svg>
  <div>
    <h4>Erro ao carregar dados</h4>
    <p>{{ errorMessage }}</p>
  </div>
  <BaseButton @click="retry">Tentar novamente</BaseButton>
</div>
```


### Success States

**Toast notification:**
```typescript
toast.success('Salvo!', 'As alterações foram salvas com sucesso')
```

**Inline feedback:**
```vue
<div v-if="saved" class="success-message">
  <svg>✓</svg>
  <span>Salvo automaticamente</span>
</div>
```

### Disabled States

```css
.button:disabled,
.input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}
```

---

## Padrões de UI Específicos

### Cards

**Card padrão:**
```css
.card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: var(--shadow-card);
}
```

**Card com header:**
```vue
<div class="card">
  <div class="card-header">
    <h3>Título do Card</h3>
    <button>Ação</button>
  </div>
  <div class="card-body">
    Conteúdo...
  </div>
</div>
```

### Tabelas

```vue
<table class="data-table">
  <thead>
    <tr>
      <th>Coluna 1</th>
      <th>Coluna 2</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Dado 1</td>
      <td>Dado 2</td>
    </tr>
  </tbody>
</table>
```

**Estilos:**
```css
.data-table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
}

.data-table th {
  background: var(--surface-2);
  color: var(--text-2);
  font-weight: 600;
  text-align: left;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--border);
}

.data-table td {
  padding: 1rem;
  border-bottom: 1px solid var(--border-soft);
}

.data-table tr:hover {
  background: var(--surface-soft);
}
```


### Modais

```vue
<Teleport to="body">
  <div v-if="isOpen" class="modal-overlay" @click="close">
    <div class="modal" @click.stop>
      <div class="modal-header">
        <h2>Título do Modal</h2>
        <button @click="close">✕</button>
      </div>
      <div class="modal-body">
        Conteúdo...
      </div>
      <div class="modal-footer">
        <BaseButton variant="ghost" @click="close">
          Cancelar
        </BaseButton>
        <BaseButton variant="primary" @click="confirm">
          Confirmar
        </BaseButton>
      </div>
    </div>
  </div>
</Teleport>
```

**Estilos:**
```css
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal {
  background: var(--surface-2);
  border-radius: 16px;
  max-width: 500px;
  width: 100%;
  max-height: 90vh;
  overflow: auto;
  box-shadow: var(--shadow-elevated);
}

.modal-header {
  padding: 1.5rem;
  border-bottom: 1px solid var(--border);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-body {
  padding: 1.5rem;
}

.modal-footer {
  padding: 1.5rem;
  border-top: 1px solid var(--border);
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
}
```


### Badges

```vue
<span class="badge badge-success">Ativo</span>
<span class="badge badge-warning">Pendente</span>
<span class="badge badge-danger">Expirado</span>
<span class="badge badge-gold">PRO</span>
```

**Estilos:**
```css
.badge {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.badge-success {
  background: var(--success-soft);
  color: var(--success);
}

.badge-warning {
  background: var(--warning-soft);
  color: var(--warning);
}

.badge-danger {
  background: var(--danger-soft);
  color: var(--danger);
}

.badge-gold {
  background: linear-gradient(135deg, #ffd700, #ffed4e);
  color: #4a3800;
  font-weight: 700;
}
```

### Tabs

```vue
<div class="tabs">
  <button 
    v-for="tab in tabs" 
    :key="tab.id"
    :class="['tab', { active: activeTab === tab.id }]"
    @click="activeTab = tab.id"
  >
    {{ tab.label }}
  </button>
</div>
```

**Estilos:**
```css
.tabs {
  display: flex;
  gap: 0.5rem;
  border-bottom: 1px solid var(--border);
}

.tab {
  padding: 0.75rem 1.25rem;
  background: transparent;
  border: none;
  color: var(--text-2);
  font-weight: 500;
  cursor: pointer;
  position: relative;
  transition: color 0.2s;
}

.tab:hover {
  color: var(--text-1);
}

.tab.active {
  color: var(--brand);
}

.tab.active::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--brand);
}
```


---

## Scrollbars Customizadas

```css
/* Tema escuro */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--text-3);
}
```

---

## Gráficos e Visualizações

### Chart.js com Tema

```typescript
// Cores adaptadas ao tema
const chartColors = {
  primary: '#1C87C1',
  success: '#22c55e',
  warning: '#f59e0b',
  danger: '#ef4444',
  text: 'var(--text-1)',
  grid: 'var(--border)',
}

const chartOptions = {
  responsive: true,
  plugins: {
    legend: {
      labels: {
        color: chartColors.text,
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.grid,
      },
      ticks: {
        color: chartColors.text,
      }
    },
    y: {
      grid: {
        color: chartColors.grid,
      },
      ticks: {
        color: chartColors.text,
      }
    }
  }
}
```


---

## Checklist de Boas Práticas

### ✅ Design Tokens
- [ ] Usar `var(--token)` ao invés de cores hardcoded
- [ ] Definir cores nos tokens, não nos componentes
- [ ] Manter consistência entre temas dark/light
- [ ] Testar ambos os temas em todos os componentes

### ✅ Responsividade
- [ ] Design mobile-first (min-width, não max-width)
- [ ] Testar em 3 breakpoints: mobile (375px), tablet (768px), desktop (1280px)
- [ ] Safe area para iOS (notch, home indicator)
- [ ] Touch targets mínimo 44x44px
- [ ] Scroll horizontal quando necessário (bottom nav)

### ✅ Acessibilidade
- [ ] Contraste mínimo 4.5:1 para texto
- [ ] Focus visible em todos elementos interativos
- [ ] Labels em todos inputs
- [ ] Alt text em todas imagens
- [ ] Navegação por teclado funcionando
- [ ] ARIA labels quando necessário

### ✅ Performance
- [ ] Animações em 60fps (transform/opacity apenas)
- [ ] Lazy loading de imagens
- [ ] Skeleton states para loading
- [ ] Debounce em buscas e inputs
- [ ] Virtual scrolling em listas longas (>100 itens)

### ✅ Consistência
- [ ] Usar componentes Base ao invés de recriar
- [ ] Espaçamentos consistentes (0.25rem, 0.5rem, 0.75rem, 1rem, 1.5rem, 2rem)
- [ ] Border radius consistente (8px cards, 12px modais, 9999px badges)
- [ ] Shadows consistentes (var(--shadow-card), var(--shadow-elevated))


---

## Estrutura de Arquivos

```
app/
├── assets/
│   └── css/
│       └── main.css              # Tokens de tema + estilos globais
│
├── components/
│   ├── Base*.vue                 # Componentes base reutilizáveis
│   ├── Sidebar.vue               # Navegação desktop
│   ├── MobileBottomNav.vue       # Navegação mobile
│   ├── AppBackground.vue         # Fundo decorativo
│   │
│   ├── admin/                    # Componentes específicos admin
│   ├── clients/                  # Componentes de clientes
│   ├── dashboard/                # Componentes do dashboard
│   └── ...
│
├── layouts/
│   ├── AppLayout.vue             # Layout autenticado
│   └── DefaultLayout.vue         # Layout público
│
├── pages/
│   ├── index.vue                 # Dashboard principal
│   ├── login.vue                 # Login
│   ├── clientes/                 # Páginas de clientes
│   └── ...
│
├── composables/
│   ├── useAuth.ts                # Auth e sessão
│   ├── useTheme.ts               # Tema dark/light
│   ├── useToast.ts               # Notificações
│   └── ...
│
└── utils/
    ├── themeColors.ts            # Helpers de cores
    └── ...

public/
├── img/
│   ├── logo-transparente.png
│   └── logo-transparente-claro.png
│
└── icons/
    ├── icon-32.png
    ├── icon-192.png
    └── icon-512.png
```


---

## Quick Start - Como Replicar Este Design

### 1. Setup Inicial

```bash
# Criar projeto Nuxt
npx nuxi init meu-projeto
cd meu-projeto

# Instalar Tailwind
npm install -D @nuxtjs/tailwindcss
```

### 2. Configurar Tailwind

```typescript
// tailwind.config.ts
export default {
  content: [
    './app/components/**/*.{vue,js,ts}',
    './app/layouts/**/*.vue',
    './app/pages/**/*.vue',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        heading: ['Instrument Sans', 'Inter', 'sans-serif']
      },
      colors: {
        primary: '#1C87C1',
        // ... outras cores
      }
    }
  }
}
```

### 3. Criar Tokens CSS

Copiar o conteúdo de `app/assets/css/main.css` deste projeto para ter todos os tokens de tema.

### 4. Componentes Base

Começar com estes componentes essenciais:
1. `BaseButton.vue`
2. `BaseInput.vue`
3. `BaseToast.vue`

### 5. Layout Base

Criar `AppLayout.vue` com:
- Sidebar (desktop)
- MobileBottomNav (mobile)
- AppBackground

### 6. Sistema de Tema

Implementar `composables/useTheme.ts` para controlar dark/light mode.


---

## Recursos Adicionais

### Ferramentas de Design
- **Figma** - Prototipagem e design system
- **Coolors.co** - Geração de paletas
- **Contrast Checker** - Verificar acessibilidade
- **Heroicons** - Biblioteca de ícones SVG

### Referências de Código
- **Tailwind CSS Docs** - https://tailwindcss.com
- **Vue 3 Composition API** - https://vuejs.org
- **Nuxt 3 Docs** - https://nuxt.com

### Inspirações de Design
- Linear (linear.app) - Design minimalista profissional
- Vercel Dashboard - Sistema de tokens bem definido
- Stripe Dashboard - Componentes e padrões consistentes

---

## Resumo dos Conceitos Principais

### 🎨 Design System
- **Tokens CSS** para cores, espaçamentos e sombras
- **Componentes base** reutilizáveis e consistentes
- **Dark/light mode** com troca dinâmica

### 📱 Responsividade
- **Mobile-first** com breakpoints progressivos
- **Sidebar** (desktop) + **Bottom Nav** (mobile)
- **Safe areas** para dispositivos modernos

### ♿ Acessibilidade
- **Contraste** adequado em ambos os temas
- **Focus states** visíveis
- **Navegação por teclado**

### ⚡ Performance
- **Animações 60fps** (transform/opacity)
- **Skeleton states** para loading
- **Lazy loading** de assets

### 🧩 Componentização
- **Atomic design** - componentes pequenos e composíveis
- **Props e slots** para flexibilidade
- **TypeScript** para type safety

---

**Última atualização:** Janeiro 2025  
**Versão do Projeto:** 2.0  
**Autor:** Documentação baseada no WhatsTracker 2.0
