package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/notify"
)

func inferDeviceLabel(ua string) string {
	lower := strings.ToLower(ua)
	osName := "Computador"
	switch {
	case strings.Contains(lower, "iphone"):
		osName = "iPhone"
	case strings.Contains(lower, "ipad"):
		osName = "iPad"
	case strings.Contains(lower, "android"):
		osName = "Android"
	case strings.Contains(lower, "macintosh") || strings.Contains(lower, "mac os"):
		osName = "Mac"
	case strings.Contains(lower, "windows"):
		osName = "Windows"
	case strings.Contains(lower, "linux"):
		osName = "Linux"
	}

	browser := "Navegador"
	switch {
	case strings.Contains(lower, "edg/"):
		browser = "Edge"
	case strings.Contains(lower, "opr/") || strings.Contains(lower, "opera"):
		browser = "Opera"
	case strings.Contains(lower, "chrome") && !strings.Contains(lower, "edg/"):
		browser = "Chrome"
	case strings.Contains(lower, "firefox"):
		browser = "Firefox"
	case strings.Contains(lower, "safari") && !strings.Contains(lower, "chrome"):
		browser = "Safari"
	}
	return osName + " • " + browser
}

func emitWhatsAppStatusNotification(svc *notify.Service, inst *domain.WhatsAppInstance, nowConnected bool) {
	if svc == nil || inst == nil {
		return
	}
	prov := notify.FriendlyWhatsAppProvider(inst.Provider)
	phone := extractPhoneFromOwnerJID(inst.OwnerJID)
	title := "WhatsApp Desconectado"
	stateVerb := "perdeu a conexão e precisa de atenção"
	stateKey := "down"
	if nowConnected {
		title = "WhatsApp Conectado"
		stateVerb = "está conectado e pronto para operar"
		stateKey = "up"
	}
	detail := fmt.Sprintf("A conexão \"%s\" (%s)", inst.Name, prov)
	if phone != "" {
		detail += " • +" + phone
	}
	svc.DispatchAsync(notify.Event{
		AccountID:   inst.AccountID,
		Category:    domain.NotifWhatsAppConnection,
		Title:       title,
		Body:        detail + " " + stateVerb + ".",
		ActionURL:   "/integracoes/whatsapp",
		CooldownKey: fmt.Sprintf("wa_conn:%d:%s", inst.ID, stateKey),
		CooldownTTL: 5 * time.Minute,
	})
}

func emitLeadNotification(svc *notify.Service, accountID int64, externalID, leadLabel, pixelName, source string) {
	if svc == nil || accountID <= 0 {
		return
	}
	who := strings.TrimSpace(leadLabel)
	if who == "" {
		who = "Novo visitante"
	}
	body := who + " entrou como novo lead"
	if source != "" {
		body += " via " + source
	}
	if pixelName != "" && pixelName != "api_import" {
		body += " • Pixel " + pixelName
	}
	body += "."
	cdKey := ""
	if externalID != "" {
		cdKey = fmt.Sprintf("lead:%d:%s", accountID, externalID)
	}
	svc.DispatchAsync(notify.Event{
		AccountID:   accountID,
		Category:    domain.NotifLeadCreated,
		Title:       "Novo Lead Recebido",
		Body:        body,
		ActionURL:   "/crm",
		CooldownKey: cdKey,
		CooldownTTL: 15 * time.Minute,
	})
}

func emitResourceLifecycleNotification(svc *notify.Service, accountID int64, category, title, body, actionURL string) {
	if svc == nil {
		return
	}
	svc.DispatchAsync(notify.Event{
		AccountID: accountID,
		Category:  category,
		Title:     title,
		Body:      body,
		ActionURL: actionURL,
	})
}

func emitCompanyLifecycleNotification(svc *notify.Service, accountID int64, companyName string, created bool) {
	if svc == nil {
		return
	}
	title := "Empresa Removida"
	body := fmt.Sprintf("A empresa \"%s\" foi removida da plataforma.", companyName)
	if created {
		title = "Nova Empresa Criada"
		body = fmt.Sprintf("A empresa \"%s\" foi criada e ativada na plataforma.", companyName)
	}
	svc.DispatchAsync(notify.Event{
		AccountID:   accountID,
		AccountName: companyName,
		Category:    domain.NotifCompanyLifecycle,
		Title:       title,
		Body:        body,
		ActionURL:   "/empresas",
		OwnersOnly:  true,
	})
}

func emitAPIKeyLifecycleNotification(svc *notify.Service, accountID int64, keyName string, created bool) {
	if svc == nil {
		return
	}
	title := "Chave de API Revogada"
	body := fmt.Sprintf("A chave de API \"%s\" foi revogada e não aceita mais requisições.", keyName)
	if created {
		title = "Nova Chave de API Criada"
		body = fmt.Sprintf("Uma nova chave de API \"%s\" foi gerada para integração externa.", keyName)
	}
	svc.DispatchAsync(notify.Event{
		AccountID: accountID,
		Category:  domain.NotifAPIKeyLifecycle,
		Title:     title,
		Body:      body,
		ActionURL: "/settings/api",
	})
}
