package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

const InFlightPurchaseTTL = 2 * time.Hour

type InFlightPurchase struct {
	EventID    string  `json:"event_id"`
	OrderID    string  `json:"order_id,omitempty"`
	IsNonOrder bool    `json:"is_non_order"`
	Value      float64 `json:"value"`
	AtUnix     int64   `json:"at_unix"`
}

// IsNonOrderPurchaseEventID identifica vendas originadas sem order_id real de gateway
// (CRM Kanban, Agente de IA, gatilho de frase, conversão manual, confirmação manual ou pixel sem order_id).
func IsNonOrderPurchaseEventID(eventID string) bool {
	id := strings.TrimSpace(eventID)
	return strings.HasPrefix(id, "crm_pur:") ||
		strings.HasPrefix(id, "crm:") ||
		strings.HasPrefix(id, "ai_sale:") ||
		strings.HasPrefix(id, "wpp_manual_") ||
		strings.HasPrefix(id, "pur_") ||
		strings.HasPrefix(id, "web_pur:") ||
		strings.HasSuffix(id, ":Purchase")
}

func IsNonOrderPurchaseRecord(eventID, orderID string) bool {
	return IsNonOrderPurchaseEventID(eventID) || strings.TrimSpace(orderID) == ""
}

// ValuesMatchCrossSource tolera diferenças de até R$ 1,00 ou até 35% entre fontes distintas
// (ex: juros de parcelamento 12x no gateway vs valor à vista no CRM/IA, desconto de PIX ou order bump).
func ValuesMatchCrossSource(a, b float64) bool {
	if a <= 0 || b <= 0 {
		return false
	}
	diff := math.Abs(a - b)
	if diff <= 1.00 {
		return true
	}
	maxVal := math.Max(a, b)
	return (diff / maxVal) <= 0.35
}

// MatchCrossSourcePurchase decide se duas compras do mesmo lead/grupo representam a mesma venda,
// preservando 100% de compras distintas quando ambos os lados possuem order_ids reais diferentes.
func MatchCrossSourcePurchase(
	existingEventID, existingOrderID string,
	existingVal float64,
	existingAt time.Time,
	incomingOrderID string,
	incomingIsNonOrder bool,
	incomingVal float64,
	aroundTime time.Time,
) bool {
	evTrim := strings.TrimSpace(existingEventID)
	exOrd := strings.TrimSpace(existingOrderID)
	inOrd := strings.TrimSpace(incomingOrderID)

	if inOrd != "" && (strings.EqualFold(evTrim, inOrd) || (exOrd != "" && strings.EqualFold(exOrd, inOrd))) {
		return true
	}

	existingIsNonOrder := IsNonOrderPurchaseRecord(evTrim, exOrd)
	if !existingIsNonOrder && !incomingIsNonOrder {
		// Dois webhooks/checkouts com order_ids reais e distintos: nunca bloquear (upsell / nova compra).
		return false
	}

	if aroundTime.IsZero() {
		aroundTime = time.Now()
	}
	diffDur := existingAt.Sub(aroundTime)
	if diffDur < 0 {
		diffDur = -diffDur
	}
	if diffDur > 72*time.Hour {
		return false
	}

	if incomingVal > 0 && existingVal > 0 {
		return ValuesMatchCrossSource(existingVal, incomingVal)
	}
	return diffDur <= 2*time.Hour
}

func purchaseIdentityKeys(accountID int64, pixelName string, groupID *int64, trackedUserID int64, externalID, phone, email string) []string {
	px := strings.TrimSpace(pixelName)
	if accountID <= 0 || px == "" {
		return nil
	}
	prefix := fmt.Sprintf("pur_inflight:%d:%s:", accountID, px)
	seen := make(map[string]bool, 5)
	out := make([]string, 0, 5)
	add := func(tok string) {
		if tok == "" {
			return
		}
		k := prefix + tok
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	if groupID != nil && *groupID > 0 {
		add(fmt.Sprintf("g:%d", *groupID))
	}
	if trackedUserID > 0 {
		add(fmt.Sprintf("u:%d", trackedUserID))
	}
	if ext := strings.TrimSpace(externalID); ext != "" {
		add("ext:" + ext)
	}
	if ph := util.NormalizePhone(phone); ph != "" {
		add("ph:" + ph)
	}
	if em := util.NormalizeEmail(email); em != "" {
		add("em:" + em)
	}
	return out
}

func RecordInFlightPurchase(
	ctx context.Context,
	rdb *redis.Client,
	accountID int64,
	pixelName string,
	groupID *int64,
	trackedUserID int64,
	externalID, phone, email string,
	rec InFlightPurchase,
) {
	if rdb == nil || strings.TrimSpace(rec.EventID) == "" {
		return
	}
	keys := purchaseIdentityKeys(accountID, pixelName, groupID, trackedUserID, externalID, phone, email)
	if len(keys) == 0 {
		return
	}
	if rec.AtUnix <= 0 {
		rec.AtUnix = time.Now().Unix()
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return
	}
	pipe := rdb.Pipeline()
	for _, k := range keys {
		pipe.HSet(ctx, k, rec.EventID, string(raw))
		pipe.Expire(ctx, k, InFlightPurchaseTTL)
	}
	_, _ = pipe.Exec(ctx)
}

func HasInFlightMatchingPurchase(
	ctx context.Context,
	rdb *redis.Client,
	accountID int64,
	pixelName string,
	groupID *int64,
	trackedUserID int64,
	externalID, phone, email string,
	incomingOrderID string,
	incomingIsNonOrder bool,
	incomingVal float64,
	aroundTime time.Time,
) bool {
	if rdb == nil {
		return false
	}
	keys := purchaseIdentityKeys(accountID, pixelName, groupID, trackedUserID, externalID, phone, email)
	if len(keys) == 0 {
		return false
	}
	pipe := rdb.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(keys))
	for i, k := range keys {
		cmds[i] = pipe.HGetAll(ctx, k)
	}
	_, _ = pipe.Exec(ctx)

	seenEv := make(map[string]bool, 4)
	for _, cmd := range cmds {
		m, err := cmd.Result()
		if err != nil {
			continue
		}
		for evID, raw := range m {
			if seenEv[evID] {
				continue
			}
			seenEv[evID] = true
			var rec InFlightPurchase
			if json.Unmarshal([]byte(raw), &rec) != nil {
				continue
			}
			exOrder := rec.OrderID
			if rec.IsNonOrder {
				exOrder = ""
			}
			if MatchCrossSourcePurchase(rec.EventID, exOrder, rec.Value, time.Unix(rec.AtUnix, 0), incomingOrderID, incomingIsNonOrder, incomingVal, aroundTime) {
				return true
			}
		}
	}
	return false
}
