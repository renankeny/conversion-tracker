package domain

// TrackerConfig é a configuração editável pelo painel que controla COMO o
// /tracker.js servido pelo backend se comporta no site do cliente.
//
// É serializada como JSONB na coluna pixels.tracker_config e injetada inline
// no JavaScript gerado para cada pixel. Mudanças refletem no próximo carregamento
// do tracker (cache curto controlado por Cache-Control).
type TrackerConfig struct {
	// Produto / checkout ----------------------------------------------------
	// ProductValue lista os preços (em moeda de ProductCurrency) atribuídos
	// aos botões de checkout encontrados na página, em ordem de aparição.
	ProductValue []float64 `json:"product_value"`
	// ProductCurrency é a moeda usada nos eventos InitiateCheckout/Purchase.
	ProductCurrency string `json:"product_currency"`
	// ProductCheckout é a substring usada pra identificar links/botões de
	// checkout (ex: "pay", "/checkout"). Se vazio, nenhum botão é interceptado.
	ProductCheckout string `json:"product_checkout"`

	// Auto-triggers (toggles) ----------------------------------------------
	// AutoPageView dispara um PageView automaticamente após o load.
	AutoPageView bool `json:"auto_page_view"`
	// AutoCheckoutButtons reescreve URLs de botões de checkout (UTMs / fbc /
	// fbp / gclid) e dispara InitiateCheckout no clique.
	AutoCheckoutButtons bool `json:"auto_checkout_buttons"`
	// AutoFormScraping captura email/phone/nome dos forms da página.
	AutoFormScraping bool `json:"auto_form_scraping"`
	// AutoFormLead dispara Lead automaticamente quando um form com email ou
	// telefone é submetido. Captura PII antes de enviar.
	AutoFormLead bool `json:"auto_form_lead"`
	// WhatsAppProtocol adiciona "Protocolo: <userId>" no parâmetro `text`
	// dos links wa.me / api.whatsapp.com — útil pra correlacionar a conversa
	// que entra no WhatsApp com o tracked_user que originou o clique.
	WhatsAppProtocol bool `json:"whatsapp_protocol"`
	// WhatsAppLead dispara o evento WhatsAppLeadEvent quando o usuário clica
	// num link de WhatsApp interceptado. Independente do WhatsAppProtocol —
	// você pode reescrever o text sem disparar o evento, ou disparar sem
	// mexer no text.
	WhatsAppLead bool `json:"whatsapp_lead"`
	// WhatsAppLeadEvent é o nome do evento disparado no clique (quando
	// WhatsAppLead=true). "Lead" ou "Contact". Default "Lead".
	WhatsAppLeadEvent string `json:"whatsapp_lead_event"`
	// SendScrolls dispara Scroll_25/50/75/90 conforme o usuário rola.
	SendScrolls bool `json:"send_scrolls"`
	// SendTimer1Min dispara Timer_1min após 60s na página.
	SendTimer1Min bool `json:"send_timer_1min"`
	// SendViewVideo dispara ViewVideo_25/50/75/90 quando há <video>.
	SendViewVideo bool `json:"send_view_video"`

	// Page rules: dispara um evento quando a URL/host casa com `Page` --------
	PageRules []PageRule `json:"page_rules"`

	// FormMappings: que atributos `name` de input olhar pra extrair PII -----
	FormMappings FormMappings `json:"form_mappings"`

	// Meta -----------------------------------------------------------------
	// MetaInitPixelID é o pixel id usado em fbq('init', ...) no client.
	// Se vazio, o tracker não inicializa o fbevents.js (cliente faria isso fora).
	MetaInitPixelID string `json:"meta_init_pixel_id"`

	// UTMOriginParam é o nome do parâmetro usado pra carregar o `userId` na
	// URL (default "utm_source"). O legado usa esse nome.
	UTMOriginParam string `json:"utm_origin_param"`
}

// PageRule mapeia "se a URL casa com Page, dispara Event (com Value/Currency
// opcionais)". Match é por substring case-insensitive.
type PageRule struct {
	Page     string  `json:"page"`
	Event    string  `json:"event"`
	Value    float64 `json:"value,omitempty"`
	Currency string  `json:"currency,omitempty"`
}

// FormMappings define quais atributos `name` de inputs casar pra cada campo
// de PII. O tracker varre todos os inputs e usa o primeiro match.
type FormMappings struct {
	Email     []string `json:"email"`
	Phone     []string `json:"phone"`
	FirstName []string `json:"first_name"`
	LastName  []string `json:"last_name"`
}

// DefaultTrackerConfig devolve uma config inicial sensata pra um pixel novo.
// O painel pode usar isso pra pré-preencher o formulário de criação.
func DefaultTrackerConfig() TrackerConfig {
	return TrackerConfig{
		ProductCurrency:     "BRL",
		UTMOriginParam:      "utm_source",
		AutoPageView:        true,
		AutoCheckoutButtons: true,
		AutoFormScraping:    true,
		AutoFormLead:        true,
		WhatsAppProtocol:    true,
		WhatsAppLead:        true,
		WhatsAppLeadEvent:   "Lead",
		FormMappings: FormMappings{
			Email:     []string{"email", "form_fields[email]"},
			Phone:     []string{"tel", "telefone", "phone", "celular", "cellphone", "form_fields[phone]"},
			FirstName: []string{"nome", "name", "first_name", "form_fields[name]", "form_fields[firstname]"},
			LastName:  []string{"sobrenome", "lastname", "last_name", "form_fields[lastname]"},
		},
		PageRules: []PageRule{},
	}
}
