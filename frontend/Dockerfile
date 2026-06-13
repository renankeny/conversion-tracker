# =============================================================================
# TrackFind Frontend - Nuxt 4 SSR
# =============================================================================
# Build multi-stage com limites de memória pra builds em VPS pequenas (2-4GB).
# Timezone obrigatória: America/Sao_Paulo (regra do projeto §9).

# ===== Stage 1: Builder =====================================================
FROM node:20-alpine AS builder

WORKDIR /app

RUN apk add --no-cache libc6-compat tzdata

# Copia package files primeiro pra aproveitar cache de layer
COPY package.json package-lock.json* ./

# --no-audit / --no-fund / --prefer-offline pra economizar tempo e memória
RUN npm ci --no-audit --no-fund --prefer-offline

# Copia o código
COPY . .

# Limita o heap do Node a 1.5GB pra build não estourar VPS pequena.
# (Nuxt + Vite + TS conseguem comer >2GB se deixar livre)
ENV NODE_ENV=production
ENV NODE_OPTIONS="--max-old-space-size=1536"

RUN npm run build

# ===== Stage 2: Runner ======================================================
FROM node:20-alpine AS runner

WORKDIR /app

RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/America/Sao_Paulo /etc/localtime && \
    echo "America/Sao_Paulo" > /etc/timezone

ENV NODE_ENV=production
ENV TZ=America/Sao_Paulo
ENV NITRO_HOST=0.0.0.0
ENV NITRO_PORT=3000

# Apenas o output do Nitro
COPY --from=builder /app/.output ./.output

EXPOSE 3000

CMD ["node", ".output/server/index.mjs"]
