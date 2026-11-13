<script setup lang="ts">
import { buildApiDocSnippets } from '~/utils/apiDocSnippets'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

const { formatTitle, brandName } = useBranding()
useHead({ title: computed(() => formatTitle('Documentação da API')) })

const config = useRuntimeConfig()
const apiUrl = computed(() => String(config.public.apiUrl || 'http://localhost:8080/api/v1'))
const snippets = computed(() => buildApiDocSnippets(apiUrl.value))
</script>

<template>
  <div class="h-full bg-background flex flex-col">
    <div class="flex-1 w-full flex flex-col lg:flex-row">
      <div class="flex-1 max-w-3xl px-6 lg:px-12 py-8 overflow-y-auto">
        <div class="mb-10">
          <h1 class="doc-h1">API Reference</h1>
          <p class="doc-lead">
            Integre seus sistemas via API. Todos os endpoints requerem uma API Key em
            <code class="code-inline">X-API-Key</code> ou <code class="code-inline">Authorization: Bearer</code>.
          </p>
        </div>

        <section class="mb-16">
          <h2 class="doc-h2">Autenticação</h2>
          <p class="doc-p">
            Para autenticar suas requisições, você precisará de uma API Key. Crie uma em <strong>Configurações &gt; API Keys</strong>.
            Passe essa chave no header das suas chamadas:
          </p>
          <div class="bg-surface-soft p-4 rounded-xl border border-border">
            <code class="doc-endpoint">X-API-Key: trk_xxxxxxxxxxxxxxxxxxxxxx</code><br /><br />
            <span class="doc-caption">Para as rotas do CRM, use o formato Bearer:</span><br />
            <code class="doc-endpoint">Authorization: Bearer trk_xxxxxxxxxxxxxxxxxxxxxx</code>
          </div>
        </section>

        <section class="mb-16">
          <h2 class="doc-h2">Envio de Eventos</h2>

          <div class="mb-12">
            <h3 class="doc-h3">Enviar Evento Único</h3>
            <div class="flex items-center gap-2 mb-3">
              <span class="badge badge-success">POST</span>
              <code class="doc-endpoint">/api/v1/send-event</code>
            </div>
            <p class="doc-p">
              Envia um evento de conversão, como Lead, Purchase ou AddToCart. O {{ brandName }} se encarrega de disparar para o Meta, TikTok e Google Ads.
            </p>

            <div class="info-callout mb-4">
              <span class="callout-icon">💡</span>
              <div>
                <strong class="block mb-1">Criação Automática no CRM:</strong>
                Se você enviar o <code>event_name: "Lead"</code> junto com o <code>external_id</code> (seu user ID), a API não apenas despachará o evento para as redes, mas <strong>criará/atualizará o contato automaticamente no Kanban do CRM</strong>.
              </div>
            </div>

            <h4 class="doc-h4">Response (202 Accepted)</h4>
            <pre class="code-box">{
  "queued": true,
  "event_id": "ABC123XYZ4"
}</pre>
          </div>

          <div>
            <h3 class="doc-h3">Enviar Eventos em Lote</h3>
            <div class="flex items-center gap-2 mb-3">
              <span class="badge badge-success">POST</span>
              <code class="doc-endpoint">/api/v1/send-events-batch</code>
            </div>
            <p class="doc-p">Envie até 100 eventos de uma vez para otimizar o processamento.</p>
          </div>
        </section>

        <section class="mb-16">
          <h2 class="doc-h2">Integração CRM</h2>

          <div class="mb-12">
            <h3 class="doc-h3">Listar Etapas</h3>
            <div class="flex items-center gap-2 mb-3">
              <span class="badge badge-info">GET</span>
              <code class="doc-endpoint">/api/v1/admin/ext-api/crm/stages</code>
            </div>
            <p class="doc-p">Lista as etapas do funil de vendas. Os IDs retornados podem ser usados na criação do lead.</p>
            <h4 class="doc-h4">Response (200 OK)</h4>
            <pre class="code-box">{
  "stages": [{ "id": 1, "name": "Lead Frio", "color": "gray" }]
}</pre>
          </div>

          <div>
            <h3 class="doc-h3">Criar ou Atualizar Lead</h3>
            <div class="flex items-center gap-2 mb-3">
              <span class="badge badge-success">POST</span>
              <code class="doc-endpoint">/api/v1/admin/ext-api/crm/leads</code>
            </div>
            <p class="doc-p">Cria um lead e insere na coluna desejada do Kanban. Se o e-mail ou telefone já existir na base, atualiza e move o lead.</p>
            <h4 class="doc-h4">Response (200 OK)</h4>
            <pre class="code-box">{
  "success": true,
  "lead_id": 45,
  "stage_id": 1
}</pre>
          </div>
        </section>

        <section class="mb-16">
          <h2 class="doc-h2">Consulta de Usuário</h2>

          <div class="mb-12">
            <h3 class="doc-h3">Lookup</h3>
            <div class="flex items-center gap-2 mb-3">
              <span class="badge badge-info">GET</span>
              <code class="doc-endpoint">/api/v1/admin/tracked-users/lookup</code>
            </div>
            <p class="doc-p">Busca um usuário já identificado pelo <code class="code-inline">email</code>, <code class="code-inline">phone</code> ou <code class="code-inline">external_id</code>.</p>
            <h4 class="doc-h4">Response (200 OK)</h4>
            <pre class="code-box">{
  "id": 42,
  "external_id": "ABC1234567",
  "first_name": "João",
  "email": "joao@exemplo.com",
  "first_utm_source": "facebook",
  "first_utm_campaign": "black-friday"
}</pre>
          </div>
        </section>
      </div>

      <div class="lg:w-[500px] xl:w-[600px] bg-surface-soft border-l border-border p-6 overflow-y-auto">
        <div class="sticky top-6 space-y-10">
          <div>
            <h4 class="snippet-title">Exemplo: Enviar Evento Único</h4>
            <CodeTabs :tabs="snippets.sendEventTabs" />
          </div>
          <div>
            <h4 class="snippet-title">Exemplo: Enviar Lote</h4>
            <CodeTabs :tabs="snippets.batchTabs" />
          </div>
          <div>
            <h4 class="snippet-title">Exemplo: Listar Etapas</h4>
            <CodeTabs :tabs="snippets.stageTabs" />
          </div>
          <div>
            <h4 class="snippet-title">Exemplo: Criar Lead no CRM</h4>
            <CodeTabs :tabs="snippets.crmLeadTabs" />
          </div>
          <div>
            <h4 class="snippet-title">Exemplo: Lookup de Usuário</h4>
            <CodeTabs :tabs="snippets.lookupTabs" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.doc-h1 { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.5rem; }
.doc-lead { font-size: var(--fs-md); color: var(--text-2); }
.doc-h2 { font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 1rem; padding-bottom: 0.5rem; border-bottom: 1px solid var(--border); }
.doc-h3 { font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.5rem; }
.doc-h4 { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.5rem; }
.doc-p { font-size: var(--fs-lg); color: var(--text-2); margin-bottom: 1rem; }
.doc-endpoint { font-family: var(--font-mono); font-size: var(--fs-lg); color: var(--text-1); }
.doc-caption { font-size: var(--fs-base); color: var(--text-3); }
.code-inline { font-family: var(--font-mono); font-size: var(--fs-base); background: var(--surface-soft); padding: 0.1rem 0.35rem; border-radius: 4px; color: var(--text-1); }
.snippet-title { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.75rem; }
.callout-icon { font-size: var(--fs-xl); }
.info-callout { background: var(--info-soft); border: 1px solid var(--border); color: var(--text-1); padding: 1rem; border-radius: 12px; font-size: var(--fs-md); display: flex; gap: 0.75rem; align-items: flex-start; }
.code-box { background: var(--surface-deep); border: 1px solid var(--border); border-radius: 12px; padding: 1rem; font-size: var(--fs-md); font-family: var(--font-mono); color: var(--text-1); overflow-x: auto; }
</style>
