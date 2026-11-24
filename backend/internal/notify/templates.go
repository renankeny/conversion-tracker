package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func FormatMoney(amount float64, currency string) string {
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		cur = "BRL"
	}
	formatted := fmt.Sprintf("%.2f", amount)
	parts := strings.Split(formatted, ".")
	intPart := parts[0]
	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	brValue := b.String() + "," + parts[1]
	switch cur {
	case "BRL":
		return "R$ " + brValue
	case "USD":
		return "US$ " + brValue
	case "EUR":
		return "€ " + brValue
	default:
		return cur + " " + brValue
	}
}

func FriendlyGatewayOrigin(gateway string) string {
	g := strings.ToLower(strings.TrimSpace(gateway))
	switch g {
	case "", "browser", "tracker", "pixel":
		return "Site / Landing Page"
	case "whatsapp", "whatsapp_ctwa":
		return "WhatsApp"
	case "whatsapp_ai":
		return "IA do WhatsApp"
	case "whatsapp_link":
		return "Link Rastreável WhatsApp"
	case "crm":
		return "CRM"
	case "api", "s2s":
		return "API Externa"
	case "onixpay":
		return "Onix Pay"
	default:
		return strings.ToUpper(g[:1]) + g[1:]
	}
}

func FriendlyWhatsAppProvider(provider string) string {
	switch domain.NormalizeWhatsAppProvider(provider) {
	case domain.WhatsAppProviderAPIPremium:
		return "API Premium"
	case domain.WhatsAppProviderUazapi:
		return "Uazapi"
	default:
		return "QR Code (Evolution Go)"
	}
}

// SendTest dispara imediatamente uma notificação real de teste para o usuário autenticado
// (tanto no histórico quanto via Web Push para o PC/Celular conectado).
func (s *Service) SendTest(ctx context.Context, userID, accountID int64, category string) (*domain.UserNotification, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("serviço de notificações indisponível")
	}
	title, body, actionURL, ok := sampleForCategory(category)
	if !ok {
		return nil, errors.New("categoria de teste inválida")
	}
	accName := s.repo.ResolveAccountName(ctx, accountID)
	kp, _ := s.EnsureKeys(ctx)
	item := s.deliverToUser(ctx, kp, userID, accountID, accName, category, title+" (Teste)", body, actionURL)
	if item == nil {
		return nil, errors.New("falha ao registrar notificação de teste")
	}
	return item, nil
}

func sampleForCategory(category string) (title, body, actionURL string, ok bool) {
	switch category {
	case domain.NotifWhatsAppConnection:
		return "WhatsApp Conectado", "Conexão Comercial Principal (API Premium) está online e recebendo conversas.", "/integracoes/whatsapp", true
	case domain.NotifLeadCreated:
		return "Novo Lead Recebido", "Mariana Costa iniciou contato via Instagram Bio • Pixel Principal.", "/crm", true
	case domain.NotifAILeadQualified:
		return "Lead Qualificado pela IA", "A IA qualificou Carlos Eduardo com alta intenção de compra na conversa do WhatsApp.", "/crm", true
	case domain.NotifSaleApproved:
		return "Nova Venda: R$ 497,00", "Origem: Hotmart • Produto: Mentoria Completa • Pixel Principal • Cliente: João Silva.", "/events", true
	case domain.NotifMetaAccountLifecycle:
		return "Conta Meta Ads Conectada", "A conta BM Principal #01 foi adicionada às integrações da empresa.", "/integracoes/meta-ads", true
	case domain.NotifGoogleAdsAccountLifecycle:
		return "Conta Google Ads Conectada", "A conta Google Ads Performance (123-456-7890) foi vinculada com sucesso.", "/integracoes/google-ads", true
	case domain.NotifPixelLifecycle:
		return "Novo Pixel Criado", "O pixel 'oferta-principal' foi configurado e já está pronto para rastrear.", "/integracoes/pixels", true
	case domain.NotifEventDispatchError:
		return "Falha no Envio de Evento", "O evento Purchase no canal Meta CAPI retornou erro de autenticação (HTTP 401).", "/events", true
	default:
		return "", "", "", false
	}
}
