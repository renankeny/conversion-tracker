# =============================================================================
# TrackFind Frontend - Nuxt 4 SSR
# =============================================================================
FROM node:20-alpine3.20 AS builder

WORKDIR /app
RUN apk add --no-cache libc6-compat tzdata
COPY package.json package-lock.json* ./
RUN npm ci --no-audit --no-fund --prefer-offline
COPY . .

ENV NODE_ENV=production
ENV NODE_OPTIONS="--max-old-space-size=1536"
RUN npm run build

FROM node:20-alpine3.20 AS runner

WORKDIR /app
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/America/Sao_Paulo /etc/localtime && \
    echo "America/Sao_Paulo" > /etc/timezone && \
    chown -R node:node /app

ENV NODE_ENV=production
ENV TZ=America/Sao_Paulo
ENV NITRO_HOST=0.0.0.0
ENV NITRO_PORT=3000

COPY --from=builder --chown=node:node /app/.output ./.output
USER node
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
