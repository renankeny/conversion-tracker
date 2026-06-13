# TrackFind - Guia de Design e Arquitetura Frontend

> **Steering Guide - Frontend Design**  
> Este documento direciona a implementação do painel Nuxt 4 do TrackFind.  
> **Documento completo:** `/TRACKFIND-DESIGN-GUIDE.md` (raiz do projeto)

---

## Referência Completa

O guia de design completo está documentado em:
📄 **`/TRACKFIND-DESIGN-GUIDE.md`**

Sempre consulte esse documento antes de:
- Criar novos componentes
- Implementar páginas
- Definir estilos ou cores
- Adicionar interações ou animações

---

## Regras Críticas (Quick Reference)

### 1. Sempre Usar Design Tokens

**✅ Correto:**
```css
background: var(--surface-2);
color: var(--text-1);
border: 1px solid var(--border);
```

**❌ Evitar:**
```css
background: #1f1f1f;  /* hardcoded */
```

### 2. Componentes Base Obrigatórios

Nunca recriar componentes que já existem em `Base*.vue`:
- `BaseButton` - botões
- `BaseInput` - inputs de form
- `BaseSelect` - dropdowns
- `BaseTable` - tabelas de dados
- `BaseModal` - modais
- `BaseBadge` - badges de status
- `BaseToast` - notificações
- `BaseEmptyState` - estados vazios

### 3. Nomenclatura de Componentes

- **Componentes Base:** `Base*.vue` (PascalCase)
- **Componentes de Domínio:** `Pixel*.vue`, `Event*.vue`, `User*.vue`
- **Layouts:** `*Layout.vue`

### 4. Mobile-First Obrigatório

Sempre escrever CSS mobile-first:
```css
/* Mobile (padrão) */
.container { width: 100%; }

/* Desktop (progressivo) */
@media (min-width: 1024px) {
  .container { max-width: 1200px; }
}
```

### 5. Responsividade - Sidebar vs Bottom Nav

- **Desktop (≥1024px):** Sidebar fixa à esquerda
- **Mobile (<1024px):** Bottom navigation com safe-area

### 6. Sempre Suportar Dark/Light

Todos os componentes devem funcionar em ambos os temas.
Testar visualmente antes de considerar completo.

### 7. Loading States Obrigatórios

Toda operação assíncrona precisa de feedback visual:
- Botões: `:loading="true"` (spinner)
- Listas: skeleton screens
- Páginas: loading overlay

### 8. Empty States Obrigatórios

Toda lista/tabela precisa de empty state:
```vue
<BaseEmptyState
  v-if="!data.length"
  title="Nenhum dado encontrado"
  description="..."
  @action="createNew"
/>
```

### 9. Máscaras de Dados Sensíveis

Sempre mascarar na UI:
- Tokens: `EAA***...***xyz`
- Emails: `us***@***.com`
- Telefones: `(11) 9****-4444`

### 10. Confirmação Dupla para Ações Destrutivas

Usar `ConfirmModal` antes de deletar:
```vue
<ConfirmModal
  :show="showDeleteConfirm"
  title="Deletar Pixel?"
  message="Esta ação não pode ser desfeita."
  confirm-text="Deletar"
  @confirm="handleDelete"
  @cancel="showDeleteConfirm = false"
/>
```

---

## Estrutura de Pastas

```
frontend/app/
├── assets/css/main.css       # Tokens CSS
├── components/
│   ├── Base*.vue             # Componentes base
│   ├── Sidebar.vue           # Navegação
│   ├── MobileBottomNav.vue
│   ├── pixels/               # Componentes de domínio
│   ├── events/
│   └── dashboard/
├── layouts/
│   ├── AppLayout.vue         # Layout autenticado
│   └── DefaultLayout.vue     # Layout público
├── pages/
│   ├── index.vue             # Dashboard
│   ├── login.vue
│   ├── pixels.vue
│   ├── events.vue
│   ├── users.vue
│   └── settings.vue
├── composables/
│   ├── useAuth.ts
│   ├── useTheme.ts
│   ├── useToast.ts
│   └── useApi.ts
└── middleware/
    └── authGuard.ts
```

---

## Stack Tecnológica

- **Nuxt 4** - Framework Vue SSR
- **Tailwind CSS 3.4** - Utility-first
- **Heroicons** - Ícones SVG
- **Chart.js** - Gráficos
- **date-fns** - Manipulação de datas

---

## APIs Backend Disponíveis

### Admin Auth
- `POST /api/v1/admin/login` - Login

### Pixels
- `GET /api/v1/admin/pixels` - Listar
- `POST /api/v1/admin/pixels` - Criar
- `PUT /api/v1/admin/pixels/:id` - Editar
- `DELETE /api/v1/admin/pixels/:id` - Deletar

### Eventos
- `GET /api/v1/admin/sent-events?pixel=&event=&status=&from=&to=&limit=&offset=`

### Usuários
- `GET /api/v1/admin/tracked-users?pixel=&search=&limit=&offset=`

### Stats
- `GET /api/v1/admin/stats?from=&to=&pixel=`

---

## Paleta de Cores TrackFind

### Marca
- **Primary (Brand):** `#1C87C1` - Azul conversion tracking
- **Primary Hover:** `#1570a3`

### Estados
- **Success:** `#22c55e` - Evento enviado
- **Danger:** `#ef4444` - Erro
- **Warning:** `#f59e0b` - Pendente/fila
- **Info:** `#06B6D4` - Informativo

### Superfícies (Dark Theme)
- `--surface-1`: `#2a2a2a` - Fundo página
- `--surface-2`: `#1f1f1f` - Cards, sidebar
- `--surface-3`: `#2a2a2a` - Hover, dropdowns

### Texto (Dark Theme)
- `--text-strong`: `#f5f5f5` - Títulos
- `--text-1`: `#e5e5e5` - Primário
- `--text-2`: `#a3a3a3` - Secundário
- `--text-3`: `#737373` - Terciário/hints

---

## Tipografia

- **Títulos:** Instrument Sans (font-heading)
- **Corpo:** Inter (font-sans)
- **Tamanhos:** text-3xl, text-2xl, text-xl, text-base, text-sm, text-xs

---

## Ícones Principais

- Dashboard: `ChartBarIcon`
- Pixels: `CursorArrowRaysIcon`
- Eventos: `PaperAirplaneIcon`
- Usuários: `UsersIcon`
- Configurações: `Cog6ToothIcon`
- Adicionar: `PlusIcon`
- Editar: `PencilIcon`
- Deletar: `TrashIcon`

---

## Checklist de PR Frontend

Antes de considerar uma feature pronta:

- [ ] Funciona em dark e light theme
- [ ] Responsivo (mobile, tablet, desktop testados)
- [ ] Loading states implementados
- [ ] Empty states implementados
- [ ] Error handling (toast + mensagens inline)
- [ ] Dados sensíveis mascarados
- [ ] Confirmação em ações destrutivas
- [ ] Navegação por teclado funcional
- [ ] Componentes Base utilizados (não recriados)
- [ ] Design tokens usados (sem hardcode)

---

**Para detalhes completos, consultar: `/TRACKFIND-DESIGN-GUIDE.md`**
- Acessar dashboard com métricas agregadas (stats)
- Gerenciar URL mappings para UTM tracking
- Configurar tracker.js personalizável por pixel

### Stack Visual
- **Tailwind CSS 3.4** - Utility-first framework
- **CSS Custom Properties** - Tokens de tema dinâmicos
- **Google Fonts** - Instrument Sans (títulos) + Inter (corpo)
- **SVG Icons** - Heroicons inline
- **Chart.js** - Gráficos de métricas e conversões

---

## Sistema de Cores e Tokens

### Arquitetura de Cores

Usamos **CSS Custom Properties** para permitir troca dinâmica de tema sem reload.

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

  /* Marca TrackFind - azul de conversão */
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
  --info: #06b6d4;
  --info-soft: rgba(6, 182, 212, 0.1);

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
  --info-soft: rgba(6, 182, 212, 0.12);

  --shadow-card: 0 1px 3px rgba(15, 23, 42, 0.08), 0 1px 2px rgba(15, 23, 42, 0.04);
  --shadow-elevated: 0 10px 32px rgba(15, 23, 42, 0.12);

  color-scheme: light;
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

**❌ Evitar** - Hardcoded:
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
**Inter** (fallback Tailwind) - Corpo de texto

### Hierarquia Tipográfica

```css
h1 { font-size: 3xl (1.875rem); font-weight: 700; }
h2 { font-size: 2xl (1.5rem); font-weight: 600; }
h3 { font-size: xl (1.25rem); font-weight: 600; }
body { font-size: base (1rem); font-weight: 400; }
.text-sm { font-size: 0.875rem; }
.text-xs { font-size: 0.75rem; }
```

### Configuração Tailwind

```typescript
// tailwind.config.ts
fontFamily: {
  sans: ['Inter', ...defaultTheme.fontFamily.sans],
  heading: ['Instrument Sans', 'Inter', ...defaultTheme.fontFamily.sans]
}
```

**Uso:**
```html
<h1 class="font-heading text-3xl font-bold">Dashboard TrackFind</h1>
<p class="font-sans text-base">Seus pixels de conversão</p>
```

---

## Componentes Base

### Arquitetura de Componentes

Todos em `app/components/` seguindo padrão `Base*.vue`:

```
app/components/
├── BaseButton.vue       # Botões com variantes
├── BaseInput.vue        # Inputs de formulário
├── BaseDropdown.vue     # Seletor dropdown
├── BaseAvatar.vue       # Avatar de usuário admin
├── BaseToast.vue        # Notificações toast
├── BaseModal.vue        # Modal genérico
├── BaseBadge.vue        # Badges de status
├── BaseTable.vue        # Tabela de dados
└── ConfirmModal.vue     # Modal de confirmação
```

### BaseButton

**Variantes:**
- `primary` - Ação principal (azul)
- `success` - Ação positiva (verde)
- `danger` - Ação destrutiva (vermelho)
- `ghost` - Botão transparente
- `outline` - Com borda

**Tamanhos:** `sm`, `md` (padrão), `lg`

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
  Salvar Pixel
</BaseButton>
```

### BaseInput

**Features:**
- Label com indicador required (*)
- Ícone à esquerda (slot)
- Estados: error, disabled, focus
- Mensagem de erro e hint

**Exemplo:**
```vue
<BaseInput
  v-model="pixelName"
  type="text"
  label="Nome do Pixel"
  placeholder="Ex: Pixel de Vendas"
  :error="errors.pixelName"
  hint="Nome amigável para identificar o pixel"
  required
/>
```

### BaseBadge

Para status de eventos e pixels:

```vue
<BaseBadge variant="success">Enviado</BaseBadge>
<BaseBadge variant="warning">Pendente</BaseBadge>
<BaseBadge variant="danger">Erro</BaseBadge>
<BaseBadge variant="info">Processing</BaseBadge>
```

### BaseTable

Tabela responsiva com sorting e paginação:

```vue
<BaseTable
  :columns="columns"
  :data="events"
  :loading="loading"
  @sort="handleSort"
>
  <template #cell-status="{ row }">
    <BaseBadge :variant="getStatusVariant(row.status)">
      {{ row.status }}
    </BaseBadge>
  </template>
</BaseTable>
```

---

## Layouts e Navegação

### Estrutura de Layouts

**2 layouts principais:**

#### 1. DefaultLayout
- Páginas públicas (login)
- Header simples
- Centro da tela, max-width 7xl

#### 2. AppLayout
- Área autenticada
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

**Estrutura:**
```
┌────────────────────┐
│  Logo TrackFind    │
├────────────────────┤
│  • Dashboard       │
│  • Pixels          │
│  • Eventos         │
│  • Usuários        │
│  • URL Mappings    │
│  • Configurações   │
├────────────────────┤
│  User Info         │
│  Logout            │
└────────────────────┘
```

**Estados do Link:**
- Normal: texto muted
- Hover: background soft, texto mais forte
- Active: background brand-soft, texto brand, borda esquerda brand

### MobileBottomNav

**Características:**
- Aparece em `max-width: 1023px`
- Fixed bottom com safe-area iOS
- Ícones 20px + label pequeno

**Layout:**
```
┌──────────────────────────────────┐
│ [🏠] [📊] [👥] [⚙️] [👤]         │
│ Home Pixels Event Conf Perf      │
└──────────────────────────────────┘
  ↑ padding-bottom: safe-area
```

---

## Sistema de Temas

### Implementação

Tema controlado por atributo `data-theme` no `<html>`:

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
setTheme('light')

// Verificar
if (isDark.value) { ... }
```

### Persistência

**3 camadas:**
1. **Cookie local** - Sobrevive refresh
2. **State global** - Reativo no Vue
3. **Banco de dados** - `admin_users.preferences` JSONB (quando logado)

---

## Responsividade e Mobile

### Breakpoints Tailwind

```typescript
screens: {
  'sm': '640px',    // Mobile grande
  'md': '768px',    // Tablet
  'lg': '1024px',   // Desktop pequeno
  'xl': '1280px',   // Desktop
  '2xl': '1536px'   // Desktop grande
}
```

### Estratégia Mobile-First

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

### Padrões Responsivos

#### Layout Principal

```css
/* Mobile: bottom nav */
@media (max-width: 1023px) {
  .sidebar { display: none; }
  .app-main { 
    margin-left: 0; 
    padding-bottom: calc(95px + env(safe-area-inset-bottom));
  }
  .mobile-bottom-nav { display: block; }
}

/* Desktop: sidebar */
@media (min-width: 1024px) {
  .sidebar { display: flex; }
  .app-main { margin-left: 280px; }
  .mobile-bottom-nav { display: none; }
}
```

#### Cards/Grid

```html
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
  <StatCard title="Eventos Enviados" :value="stats.total_events" />
  <StatCard title="Taxa de Sucesso" :value="stats.success_rate" />
  <StatCard title="Usuários Rastreados" :value="stats.tracked_users" />
</div>
```

### Safe Area (iOS)

```css
/* Top padding para notch */
padding-top: calc(2rem + env(safe-area-inset-top, 0));

/* Bottom padding para indicador */
padding-bottom: calc(95px + env(safe-area-inset-bottom, 0));
```

### Touch Targets

**Mínimo:** 44x44px (Apple HIG)

```css
min-height: 44px;
min-width: 44px;
```

---

## Padrões de Interação

### Animações e Transições

**Princípio:** Suave mas perceptível (200-300ms)

```css
/* Hover states */
transition: all 0.2s ease;

/* Backgrounds */
transition: background 0.15s, border-color 0.15s;

/* Transformações */
transition: transform 0.18s;

/* Opacidade */
transition: opacity 0.2s ease;
```

### Animações Específicas

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

### Hover States

```css
.button:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.card:hover {
  box-shadow: var(--shadow-elevated);
  transform: translateY(-2px);
}
```

### Focus States

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

Quando não há dados:

```vue
<div class="empty-state">
  <svg class="empty-icon">...</svg>
  <h3 class="empty-title">Nenhum pixel cadastrado</h3>
  <p class="empty-description">
    Crie seu primeiro pixel para começar a rastrear conversões.
  </p>
  <BaseButton variant="primary" @click="createPixel">
    Criar Pixel
  </BaseButton>
</div>
```

### Error States

**Inline (formulários):**
```vue
<BaseInput
  v-model="email"
  :error="errors.email"
/>
```

**Page-level:**
```vue
<div class="error-banner">
  <svg class="error-icon">❌</svg>
  <div>
    <h4>Erro ao carregar eventos</h4>
    <p>{{ errorMessage }}</p>
  </div>
  <BaseButton @click="retry">Tentar novamente</BaseButton>
</div>
```

### Success States

```typescript
const toast = useToast()
toast.success('Pixel salvo!', 'Configurações atualizadas com sucesso')
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

## Padrões de UI Específicos TrackFind

### Cards de Estatísticas

```vue
<StatCard 
  title="Eventos Hoje"
  :value="stats.events_today"
  :change="+12.5"
  icon="chart"
  trend="up"
/>
```

### Tabela de Eventos

Colunas principais:
- ID do evento
- Pixel
- Tipo de evento (PageView, Purchase, etc)
- Status (sent, pending, failed)
- External ID / ID Wpp
- Timestamp
- Ações (reenviar, detalhes)

### Formulário de Pixel

Seções:
1. **Informações Básicas** - Nome, domínios permitidos
2. **Meta CAPI** - Pixel ID, Access Token
3. **Configuração do Tracker** - tracker_config JSONB editável
4. **Testes** - Enviar evento de teste

### Dashboard Métricas

- Gráfico de linha: eventos por dia (últimos 30d)
- Cards com métricas: total eventos, taxa sucesso, usuários rastreados
- Breakdown por status (sent, failed, pending)
- Breakdown por tipo de evento (PageView, Purchase, Lead, etc)

---

## Estrutura de Arquivos Frontend

```
frontend/
├── app/
│   ├── assets/
│   │   └── css/
│   │       └── main.css              # Tokens de tema
│   │
│   ├── components/
│   │   ├── Base*.vue                 # Componentes base
│   │   ├── Sidebar.vue
│   │   ├── MobileBottomNav.vue
│   │   ├── AppBackground.vue
│   │   │
│   │   ├── dashboard/
│   │   │   ├── StatCard.vue
│   │   │   └── EventsChart.vue
│   │   │
│   │   ├── pixels/
│   │   │   ├── PixelForm.vue
│   │   │   ├── PixelCard.vue
│   │   │   └── TrackerConfigEditor.vue
│   │   │
│   │   ├── events/
│   │   │   ├── EventsTable.vue
│   │   │   └── EventDetailsModal.vue
│   │   │
│   │   └── users/
│   │       └── TrackedUsersTable.vue
│   │
│   ├── layouts/
│   │   ├── AppLayout.vue             # Layout autenticado
│   │   └── DefaultLayout.vue         # Layout público
│   │
│   ├── pages/
│   │   ├── index.vue                 # Dashboard
│   │   ├── login.vue
│   │   │
│   │   ├── pixels/
│   │   │   ├── index.vue             # Lista
│   │   │   ├── create.vue
│   │   │   └── [id].vue              # Editar
│   │   │
│   │   ├── events/
│   │   │   └── index.vue
│   │   │
│   │   ├── users/
│   │   │   └── index.vue
│   │   │
│   │   ├── mappings/
│   │   │   └── index.vue
│   │   │
│   │   └── settings/
│   │       └── index.vue
│   │
│   ├── composables/
│   │   ├── useAuth.ts
│   │   ├── useTheme.ts
│   │   ├── useToast.ts
│   │   ├── usePixels.ts              # CRUD de pixels
│   │   ├── useEvents.ts              # Busca de eventos
│   │   ├── useStats.ts               # Dashboard stats
│   │   └── useAPI.ts                 # Cliente HTTP base
│   │
│   ├── middleware/
│   │   ├── auth.ts                   # Guard de autenticação
│   │   └── guest.ts                  # Redireciona logado
│   │
│   ├── utils/
│   │   ├── formatDate.ts
│   │   ├── formatNumber.ts
│   │   ├── maskToken.ts
│   │   └── validators.ts
│   │
│   ├── app.vue
│   ├── app.config.ts
│   └── error.vue
│
├── shared/
│   ├── types/
│   │   ├── Pixel.ts
│   │   ├── Event.ts
│   │   ├── TrackedUser.ts
│   │   ├── Stats.ts
│   │   └── API.ts                    # Tipos de resposta da API
│   │
│   └── constants/
│       ├── eventTypes.ts             # PageView, Purchase, Lead, etc
│       └── statusTypes.ts            # sent, pending, failed
│
├── public/
│   └── img/
│       ├── logo-dark.svg
│       ├── logo-light.svg
│       └── favicon.ico
│
├── tailwind.config.ts
├── nuxt.config.ts
└── .env
```

---

## Integração com Backend Go

### Cliente API Base

```typescript
// app/composables/useAPI.ts
export const useAPI = () => {
  const config = useRuntimeConfig()
  const auth = useAuth()

  const apiBase = config.public.apiUrl || 'http://localhost:8080/api/v1'

  const request = async <T>(
    endpoint: string,
    options: FetchOptions = {}
  ): Promise<T> => {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    }

    if (auth.token.value) {
      headers['Authorization'] = `Bearer ${auth.token.value}`
    }

    const response = await $fetch<T>(`${apiBase}${endpoint}`, {
      ...options,
      headers,
    })

    return response
  }

  return { request, apiBase }
}
```

### Endpoints Principais

```typescript
// Pixels
GET    /admin/pixels
POST   /admin/pixels
GET    /admin/pixels/:id
PUT    /admin/pixels/:id
DELETE /admin/pixels/:id

// Eventos
GET    /admin/sent-events?pixel=&event=&status=&from=&to=&limit=&offset=

// Usuários Rastreados
GET    /admin/tracked-users?pixel=&search=&limit=&offset=

// Stats
GET    /admin/stats?from=&to=&pixel=

// URL Mappings
GET    /admin/url-mappings
POST   /admin/url-mappings
DELETE /admin/url-mappings/:id

// Auth
POST   /admin/login
POST   /admin/logout
```

### Tipos TypeScript (exemplo)

```typescript
// shared/types/Pixel.ts
export interface Pixel {
  id: number
  name: string
  pixel_id: string
  access_token: string
  domains: string[]
  tracker_config: TrackerConfig
  active: boolean
  created_at: string
  updated_at: string
}

export interface TrackerConfig {
  events?: {
    page_view?: boolean
    view_content?: boolean
    add_to_cart?: boolean
    initiate_checkout?: boolean
    purchase?: boolean
    lead?: boolean
  }
  scroll_tracking?: {
    enabled?: boolean
    thresholds?: number[]
  }
  timer_tracking?: {
    enabled?: boolean
    intervals?: number[]
  }
  // ... outros campos do tracker_config
}

// shared/types/Event.ts
export interface SentEvent {
  id: number
  pixel_name: string
  event_name: string
  external_id: string
  id_wpp: string
  status: 'sent' | 'pending' | 'failed'
  error: string
  response_body: string
  created_at: string
  sent_at: string | null
}

// shared/types/Stats.ts
export interface DashboardStats {
  status_totals: {
    sent: number
    pending: number
    failed: number
  }
  event_totals: Record<string, number>
  per_day: Array<{
    date: string
    count: number
  }>
  total_events: number
  tracked_users: number
}
```

---

## Checklist de Implementação

### ✅ Setup Inicial
- [ ] Criar projeto Nuxt 4
- [ ] Instalar Tailwind CSS
- [ ] Configurar fontes (Instrument Sans + Inter)
- [ ] Criar `main.css` com tokens de tema
- [ ] Configurar `.env` com `API_URL`

### ✅ Componentes Base
- [ ] BaseButton
- [ ] BaseInput
- [ ] BaseDropdown
- [ ] BaseModal
- [ ] BaseBadge
- [ ] BaseTable
- [ ] BaseToast
- [ ] BaseAvatar

### ✅ Layouts
- [ ] DefaultLayout (público)
- [ ] AppLayout (autenticado)
- [ ] Sidebar (desktop)
- [ ] MobileBottomNav
- [ ] AppBackground

### ✅ Composables
- [ ] useAuth (login, logout, token)
- [ ] useTheme (dark/light)
- [ ] useToast (notificações)
- [ ] useAPI (cliente HTTP)
- [ ] usePixels (CRUD)
- [ ] useEvents (listagem, filtros)
- [ ] useStats (dashboard)

### ✅ Páginas
- [ ] Login (`/login`)
- [ ] Dashboard (`/`)
- [ ] Lista de Pixels (`/pixels`)
- [ ] Criar Pixel (`/pixels/create`)
- [ ] Editar Pixel (`/pixels/[id]`)
- [ ] Eventos (`/events`)
- [ ] Usuários Rastreados (`/users`)
- [ ] URL Mappings (`/mappings`)
- [ ] Configurações (`/settings`)

### ✅ Tipos TypeScript
- [ ] Pixel
- [ ] TrackerConfig
- [ ] SentEvent
- [ ] TrackedUser
- [ ] DashboardStats
- [ ] URLMapping
- [ ] API responses

### ✅ Testes
- [ ] Responsividade (mobile, tablet, desktop)
- [ ] Tema claro e escuro
- [ ] Navegação por teclado
- [ ] Loading states
- [ ] Empty states
- [ ] Error handling

---

## Referências

- **Tailwind CSS** - https://tailwindcss.com
- **Nuxt 3/4** - https://nuxt.com
- **Heroicons** - https://heroicons.com
- **Chart.js** - https://www.chartjs.org

---

## Resumo dos Princípios

### 🎨 Design System
- Tokens CSS para tudo
- Componentes base reutilizáveis
- Dark/light mode dinâmico
- Consistência visual

### 📱 Responsividade
- Mobile-first sempre
- Sidebar (desktop) + Bottom Nav (mobile)
- Safe areas para iOS
- Touch targets 44x44px

### ♿ Acessibilidade
- Contraste adequado
- Focus states visíveis
- Labels em inputs
- Navegação por teclado

### ⚡ Performance
- Animações 60fps
- Skeleton loading
- Lazy loading
- Debounce em buscas

### 🔗 Integração Backend
- Cliente HTTP tipado
- Autenticação JWT
- Error handling consistente
- Estados loading/error/success

---

**Última atualização:** Junho 2026  
**Projeto:** TrackFind - Conversion Tracking Platform
