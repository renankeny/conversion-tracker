# =============================================================================
# TrackFind Frontend - Nuxt 4 SSR
# =============================================================================
# Build multi-stage:
#   1) builder: instala deps e gera o output do Nuxt
#   2) runner:  imagem mínima com só o necessário pra rodar o Nitro server
#
# Timezone obrigatória: America/Sao_Paulo (regra do projeto §9).

# ===== Stage 1: Builder =====================================================
FROM node:20-alpine AS builder

WORKDIR /app

# Instala dependências de sistema (necessárias pra alguns pacotes do Nuxt)
RUN apk add --no-cache libc6-compat tzdata

# Copia package files primeiro pra aproveitar cache de layer
COPY package.json package-lock.json* ./

RUN npm ci --no-audit --no-fund

# Copia o resto do código
COPY . .

# Gera build de produção (Nitro output em .output/)
ENV NODE_ENV=production
RUN npm run build


# ===== Stage 2: Runner ======================================================
FROM node:20-alpine AS runner

WORKDIR /app

# Timezone America/Sao_Paulo
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/America/Sao_Paulo /etc/localtime && \
    echo "America/Sao_Paulo" > /etc/timezone

ENV NODE_ENV=production
ENV TZ=America/Sao_Paulo
ENV NITRO_HOST=0.0.0.0
ENV NITRO_PORT=3000

# Copia somente o output do build (sem node_modules, sem source)
COPY --from=builder /app/.output ./.output

EXPOSE 3000

# O Nitro server bundle inclui as deps necessárias.
CMD ["node", ".output/server/index.mjs"]
