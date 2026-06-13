#!/usr/bin/env bash
#
# update-geoip.sh — baixa a versão mais recente da base GeoLite2-City da
# MaxMind e substitui backend/data/GeoLite2-City.mmdb. Lê as credenciais
# do .env na raiz do projeto.
#
# Uso:
#   ./scripts/update-geoip.sh
#
# Variáveis lidas do .env:
#   MAXMIND_LICENSE_KEY   (obrigatória)
#   MAXMIND_ACCOUNT_ID    (opcional, informativa apenas neste fluxo)
#
# Sai com erro se a license key não estiver definida ou o checksum não bater.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${ROOT}/.env"
DEST="${ROOT}/backend/data/GeoLite2-City.mmdb"
EDITION_ID="GeoLite2-City"

log()  { printf '\033[1;34m›\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
err()  { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; }

# --- 1. Carregar .env ------------------------------------------------------

if [ ! -f "$ENV_FILE" ]; then
  err ".env não encontrado em $ENV_FILE"
  exit 1
fi

# Carrega apenas as vars que precisamos (sem poluir o shell com tudo do .env).
MAXMIND_LICENSE_KEY="$(grep -E '^MAXMIND_LICENSE_KEY=' "$ENV_FILE" | head -n1 | cut -d'=' -f2- | tr -d '"' | tr -d "'")"

if [ -z "${MAXMIND_LICENSE_KEY:-}" ]; then
  err "MAXMIND_LICENSE_KEY ausente ou vazia em $ENV_FILE"
  exit 1
fi

# --- 2. Preparar tmp + endpoints ------------------------------------------

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

ARCHIVE="${TMP_DIR}/${EDITION_ID}.tar.gz"
SHA_FILE="${ARCHIVE}.sha256"

URL="https://download.maxmind.com/app/geoip_download?edition_id=${EDITION_ID}&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
URL_SHA="${URL}.sha256"

# --- 3. Baixar archive + checksum -----------------------------------------

log "Baixando ${EDITION_ID}.tar.gz da MaxMind..."
if ! curl --fail --silent --show-error --location --output "$ARCHIVE" "$URL"; then
  err "Falha ao baixar a base. Verifique se MAXMIND_LICENSE_KEY está válida."
  exit 1
fi

log "Baixando checksum..."
if ! curl --fail --silent --show-error --location --output "$SHA_FILE" "$URL_SHA"; then
  err "Falha ao baixar checksum."
  exit 1
fi

# --- 4. Validar SHA256 ----------------------------------------------------

EXPECTED_SUM="$(awk '{print $1}' "$SHA_FILE")"
if [ -z "$EXPECTED_SUM" ]; then
  err "Checksum vazio no arquivo recebido."
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL_SUM="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL_SUM="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
else
  err "Nenhum utilitário sha256 disponível (sha256sum ou shasum)."
  exit 1
fi

if [ "$EXPECTED_SUM" != "$ACTUAL_SUM" ]; then
  err "Checksum NÃO bate. Download corrompido ou adulterado."
  err "  esperado: $EXPECTED_SUM"
  err "  obtido:   $ACTUAL_SUM"
  exit 1
fi
ok "Checksum SHA256 válido"

# --- 5. Extrair e mover atomicamente --------------------------------------

log "Extraindo arquivo..."
tar -xzf "$ARCHIVE" -C "$TMP_DIR"

# A MaxMind extrai numa pasta com data, ex: GeoLite2-City_20260613/GeoLite2-City.mmdb
EXTRACTED="$(find "$TMP_DIR" -type f -name "${EDITION_ID}.mmdb" -print -quit)"
if [ -z "$EXTRACTED" ]; then
  err "Arquivo ${EDITION_ID}.mmdb não encontrado dentro do archive."
  exit 1
fi

mkdir -p "$(dirname "$DEST")"

# Backup do anterior (caso exista) pra rollback simples se algo der errado.
if [ -f "$DEST" ]; then
  cp "$DEST" "${DEST}.bak"
fi

# Move atomicamente (mesmo filesystem dentro do tmp ou dentro do projeto).
mv "$EXTRACTED" "$DEST.new"
mv "$DEST.new" "$DEST"

SIZE="$(du -h "$DEST" | cut -f1)"
ok "Atualizado: $DEST ($SIZE)"
log "Backup do anterior em: ${DEST}.bak (apague quando quiser)"
