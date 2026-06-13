# Quickstart — Conversion Tracker

Smoke test em ~5 minutos: subir os containers, criar um pixel, embeddar o tracker numa página local e ver os eventos chegando.

---

## 1. Pré-requisitos

- Docker e Docker Compose instalados
- Um pixel real do Meta (pixel_id + access_token) com `test_event_code` ativo (Events Manager → Test Events)

---

## 2. Configurar o `.env`

```bash
cp .env.example .env
```

Abra `.env` e troque AT LEAST:

- `JWT_SECRET` — gere com `openssl rand -hex 64`
- `ADMIN_PASSWORD` — senha que você vai usar pra logar no painel
- `MAXMIND_LICENSE_KEY` — opcional, só pra rodar o `update-geoip.sh` no futuro

---

## 3. Subir os containers

```bash
docker compose up --build
```

Aguarde ver:
```
api      | api starting addr=":8080" strict_cors=false
worker   | worker starting redis="redis:6379"
postgres | database system is ready to accept connections
```

Cheque o health:
```bash
curl http://localhost:8080/api/v1/health
# {"status":"ok"}
```

---

## 4. Criar um pixel pelo admin

```bash
# Login (usa ADMIN_EMAIL / ADMIN_PASSWORD do .env)
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"SUA_SENHA"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["access_token"])')

echo "TOKEN=$TOKEN"

# Cria pixel
curl -s -X POST http://localhost:8080/api/v1/admin/pixels \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "teste",
    "label": "Pixel de teste",
    "pixel_ids": ["1234567890"],
    "access_tokens": ["EAAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"],
    "test_codes": ["TEST12345"],
    "domains": ["localhost", "127.0.0.1"]
  }' | python3 -m json.tool
```

> Pra ver a config inicial sugerida: `GET /api/v1/admin/pixels/defaults` (com Bearer).

---

## 5. Servir a página de teste

`examples/test.html` já está pronto. Use qualquer servidor estático local:

```bash
cd examples
python3 -m http.server 3000
```

Abra http://localhost:3000/test.html no navegador.

---

## 6. O que observar

**No DevTools → Network:**
- `GET /api/v1/loader.js?pixel=teste` → 200 com o stub
- `GET /api/v1/tracker.js?pixel=teste` → 200 com o JS grande + config inline
- `POST /api/v1/events/send` (eventType=Init) → 200 com geoip
- `POST /api/v1/events/send` (eventType=PageView) → 202 com event_id

**Nos logs do worker:**
```
{"level":"INFO","msg":"meta send ok","pixel_id":"1234567890","event":"PageView","event_id":"...","status":200}
```

**No Events Manager → Test Events:** os eventos `PageView`, `Lead`, `Purchase`, etc. devem aparecer com o `test_event_code` configurado.

**Nos botões da página:**
- "Lead", "Contact", "Purchase R$99,90" disparam eventos manuais via `tf('send', ...)`.
- "Comprar agora" reescreve a URL com UTM/userId/fbc/fbp/gclid e dispara `InitiateCheckout`.
- "Falar no WhatsApp" adiciona "Protocolo: USERID" no `?text=` e dispara `Lead`.

---

## 7. Eventos no banco

```bash
docker compose exec postgres psql -U tracker -d tracker -c \
  "SELECT event_id, event_name, status, response_code FROM sent_events ORDER BY id DESC LIMIT 10;"
```

```bash
docker compose exec postgres psql -U tracker -d tracker -c \
  "SELECT external_id, pixel_name, country, state, city, email, phone FROM tracked_users ORDER BY id DESC LIMIT 5;"
```

---

## 8. Testar webhook (Kiwify ou Hotmart)

```bash
curl -s -X POST http://localhost:8080/api/v1/webhooks/kiwify \
  -H 'Content-Type: application/json' \
  -d '{
    "order_id": "FAKE-001",
    "Customer": {"full_name": "Fulano da Silva", "email": "fulano@example.com", "mobile": "11999999999"},
    "Commissions": {"currency": "BRL", "my_commission": 9990},
    "TrackingParameters": {"utm_source": "fb|userIdABCDEFGHIJ|"}
  }'
```

Se houver um `tracked_user` com `external_id="ABCDEFGHIJ"` (gerado pelo tracker), ele encontra. Senão, faz fallback por phone.

---

## 9. Troubleshooting

- **`ERR_CONNECTION_REFUSED` no fetch:** API não subiu. `docker compose logs api`.
- **`pixel disabled` no /tracker.js:** crie pixel via admin (passo 4) ou ative o existente.
- **Eventos não aparecem no Meta:** confira `pixel_ids[]` e `access_tokens[]` reais; veja `sent_events.error` no Postgres.
- **CORS bloqueando:** em dev `STRICT_CORS=false` aceita tudo. Em prod, adicione o domínio do site em `pixels.domains`.
- **GeoIP retornando vazio:** sem o `.mmdb`, o tracker continua funcionando — só não enriquece país/cidade. Rode `./scripts/update-geoip.sh`.
