# TrackFind Frontend - Implementação Inicial

## ✅ O que foi implementado

### 1. Configuração Base
- ✅ Projeto Nuxt 4 criado (minimal template)
- ✅ Tailwind CSS configurado com tokens personalizados
- ✅ Fontes Google (Instrument Sans + Inter)
- ✅ Configuração de variáveis de ambiente (.env)
- ✅ TypeScript configurado

### 2. Sistema de Cores e Temas
- ✅ Tokens CSS para dark e light mode em `app/assets/css/main.css`
- ✅ Composable `useTheme` para alternar temas
- ✅ Tema escuro como padrão
- ✅ Persistência do tema em cookie

### 3. Tipos TypeScript (`shared/types/`)
- ✅ `Pixel.ts` - Interface de pixels e configuração do tracker
- ✅ `Event.ts` - Eventos enviados e filtros
- ✅ `TrackedUser.ts` - Usuários rastreados
- ✅ `Stats.ts` - Estatísticas do dashboard
- ✅ `Auth.ts` - Autenticação e usuário admin
- ✅ `API.ts` - Tipos de resposta da API

### 4. Constantes (`shared/constants/`)
- ✅ `eventTypes.ts` - Tipos de eventos Meta CAPI
- ✅ `statusTypes.ts` - Status de eventos (sent, pending, failed)

### 5. Composables (`app/composables/`)
- ✅ `useAPI.ts` - Cliente HTTP para comunicação com backend Go
- ✅ `useAuth.ts` - Autenticação (login, logout, token JWT)
- ✅ `useTheme.ts` - Controle de tema dark/light
- ✅ `useToast.ts` - Sistema de notificações toast

### 6. Componentes Base (`app/components/`)
- ✅ `BaseButton.vue` - Botão com variantes e loading
- ✅ `BaseInput.vue` - Input com label, ícone, error e hint
- ✅ `BaseBadge.vue` - Badge de status
- ✅ `BaseToast.vue` - Toast de notificação

### 7. Layouts (`app/layouts/`)
- ✅ `default.vue` - Layout simples para páginas públicas

### 8. Páginas (`app/pages/`)
- ✅ `login.vue` - Página de login funcional
- ✅ `index.vue` - Dashboard com estatísticas (requer autenticação)

### 9. Middlewares (`app/middleware/`)
- ✅ `auth.ts` - Protege rotas autenticadas
- ✅ `guest.ts` - Redireciona usuários logados

### 10. Sistema de Toasts
- ✅ Container de toasts no `app.vue`
- ✅ Animações de entrada/saída
- ✅ Auto-dismiss após 5 segundos
- ✅ Máximo de toasts empilhados

---

## 🚀 Como Usar

### Iniciar o servidor de desenvolvimento
```bash
cd frontend
npm run dev
```

O servidor estará disponível em: `http://localhost:3000`

### Credenciais de teste (backend)
```
Email: admin@example.com
Senha: troque-essa-senha
```

---

## 📂 Estrutura de Arquivos

```
frontend/
├── app/
│   ├── assets/
│   │   └── css/
│   │       └── main.css          # Tokens de tema
│   ├── components/
│   │   ├── BaseButton.vue
│   │   ├── BaseInput.vue
│   │   ├── BaseBadge.vue
│   │   └── BaseToast.vue
│   ├── composables/
│   │   ├── useAPI.ts
│   │   ├── useAuth.ts
│   │   ├── useTheme.ts
│   │   └── useToast.ts
│   ├── layouts/
│   │   └── default.vue
│   ├── middleware/
│   │   ├── auth.ts
│   │   └── guest.ts
│   ├── pages/
│   │   ├── index.vue             # Dashboard
│   │   └── login.vue
│   └── app.vue
├── shared/
│   ├── types/
│   │   ├── Pixel.ts
│   │   ├── Event.ts
│   │   ├── TrackedUser.ts
│   │   ├── Stats.ts
│   │   ├── Auth.ts
│   │   ├── API.ts
│   │   └── index.ts
│   └── constants/
│       ├── eventTypes.ts
│       └── statusTypes.ts
├── public/
│   ├── img/
│   └── favicon.ico
├── .env
├── nuxt.config.ts
├── tailwind.config.ts
└── package.json
```

---

## 🎨 Sistema de Design

### Cores (CSS Variables)

**Superfícies:**
- `--surface-1` - Fundo da página
- `--surface-2` - Cards, modais
- `--surface-3` - Hover, dropdowns

**Texto:**
- `--text-strong` - Títulos principais
- `--text-1` - Texto primário
- `--text-2` - Texto secundário
- `--text-3` - Texto terciário

**Marca:**
- `--brand` - Azul primário (#1C87C1)
- `--brand-soft` - Fundo suave
- `--brand-border` - Bordas

**Estados:**
- `--success` - Verde (#22c55e)
- `--danger` - Vermelho (#ef4444)
- `--warning` - Laranja (#f59e0b)
- `--info` - Ciano (#06b6d4)

### Componentes

**BaseButton:**
```vue
<BaseButton variant="primary" size="md" :loading="false">
  Texto do Botão
</BaseButton>
```
- Variantes: `primary`, `success`, `danger`, `ghost`, `outline`
- Tamanhos: `sm`, `md`, `lg`

**BaseInput:**
```vue
<BaseInput
  v-model="value"
  label="Label"
  placeholder="Placeholder"
  :error="errorMessage"
  hint="Dica opcional"
/>
```

**BaseBadge:**
```vue
<BaseBadge variant="success">Enviado</BaseBadge>
```
- Variantes: `success`, `warning`, `danger`, `info`, `default`

---

## 🔗 Integração com Backend

### URL da API
Configurada em `.env`:
```
NUXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

### Endpoints Utilizados

**Autenticação:**
- `POST /admin/login` - Login
- `POST /admin/logout` - Logout

**Dashboard:**
- `GET /admin/stats` - Estatísticas gerais

### Uso do cliente API

```typescript
const api = useAPI()

// GET
const data = await api.get<ResponseType>('/endpoint')

// POST
const result = await api.post<ResponseType>('/endpoint', { body })

// PUT
const updated = await api.put<ResponseType>('/endpoint/:id', { body })

// DELETE
await api.delete('/endpoint/:id')
```

---

## 🔒 Autenticação

O sistema usa JWT armazenado em cookie:

```typescript
const { login, logout, isAuthenticated, user } = useAuth()

// Login
await login({ email: 'admin@example.com', password: 'senha' })

// Logout
await logout()

// Verificar autenticação
if (isAuthenticated.value) {
  console.log('Usuário:', user.value)
}
```

---

## 📱 Responsividade

- **Mobile-first** - CSS escrito para mobile primeiro
- **Breakpoints Tailwind:**
  - `sm`: 640px
  - `md`: 768px
  - `lg`: 1024px
  - `xl`: 1280px
  - `2xl`: 1536px

---

## 🎯 Próximos Passos

### Componentes Faltantes
- [ ] BaseDropdown - Seletor com busca
- [ ] BaseModal - Modal genérico
- [ ] BaseTable - Tabela de dados
- [ ] BaseAvatar - Avatar de usuário
- [ ] Sidebar - Navegação desktop
- [ ] MobileBottomNav - Navegação mobile
- [ ] AppBackground - Fundo decorativo

### Páginas Faltantes
- [ ] `/pixels` - Lista de pixels
- [ ] `/pixels/create` - Criar pixel
- [ ] `/pixels/[id]` - Editar pixel
- [ ] `/events` - Lista de eventos
- [ ] `/users` - Usuários rastreados
- [ ] `/mappings` - URL mappings
- [ ] `/settings` - Configurações

### Composables Faltantes
- [ ] `usePixels` - CRUD de pixels
- [ ] `useEvents` - Listagem de eventos
- [ ] `useStats` - Carregar estatísticas

### Funcionalidades
- [ ] AppLayout com sidebar + bottom nav
- [ ] Gráficos com Chart.js
- [ ] Filtros avançados nas tabelas
- [ ] Paginação
- [ ] Editor de tracker_config JSONB
- [ ] Testes com evento de teste

---

## 🐛 Debug

### Ver output do servidor
```bash
# No terminal onde rodou npm run dev
```

### Verificar erros no console
- Abra o navegador em `http://localhost:3000`
- F12 para abrir DevTools
- Console e Network tabs

### Testar API
```bash
# Login
curl -X POST http://localhost:8080/api/v1/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"troque-essa-senha"}'
```

---

## 📚 Referências

- [Nuxt 4 Docs](https://nuxt.com)
- [Tailwind CSS](https://tailwindcss.com)
- [Vue 3 Composition API](https://vuejs.org)
- [Guia de Design do Projeto](.kiro/steering/frontend-design-guide.md)
