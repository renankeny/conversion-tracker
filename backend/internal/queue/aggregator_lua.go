package queue

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// Script Lua atômico para agregar compras (main + order_bump + upsell) em 1 round-trip,
// eliminando spinlocks (SetNX + Sleep) e condições de corrida entre workers/réplicas.
var aggregatePurchaseScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
local ttl = tonumber(ARGV[9])
if not raw or raw == '' then
  redis.call('SET', KEYS[1], ARGV[1], 'EX', ttl)
  return {1, 0, ARGV[1]}
end

local ok, session = pcall(cjson.decode, raw)
if not ok or type(session) ~= 'table' then
  redis.call('SET', KEYS[1], ARGV[1], 'EX', ttl)
  return {1, 0, ARGV[1]}
end

local item = cjson.decode(ARGV[2])
local pii = cjson.decode(ARGV[3])
local orderID = ARGV[4]
local itemType = ARGV[5]
local contentID = ARGV[6]
local contentName = ARGV[7]
local val = tonumber(ARGV[8]) or 0

local function hasValue(list, target)
  if type(list) ~= 'table' then return false end
  for i = 1, #list do
    if list[i] == target then return true end
  end
  return false
end

local function mergePII(s, p)
  if (not s.external_id or s.external_id == '') and p.external_id ~= '' then s.external_id = p.external_id end
  if (not s.source_url or s.source_url == '') and p.source_url ~= '' then s.source_url = p.source_url end
  local preferIncoming = {
    'email','phone','first_name','last_name','country','state','city','postal_code',
    'fbp','fbc','gclid','wbraid','gbraid','ttclid','ttp','ga4_client_id','ga4_session_id',
    'client_ip','client_user_agent'
  }
  for _, f in ipairs(preferIncoming) do
    if p[f] and p[f] ~= '' then s[f] = p[f] end
  end
end

local function cleanEmptyArrays(s)
  local arrFields = {'order_ids', 'content_ids', 'content_names', 'items'}
  for _, f in ipairs(arrFields) do
    if type(s[f]) ~= 'table' or next(s[f]) == nil then
      s[f] = nil
    end
  end
end

local isBumpOrUpsell = (itemType == 'order_bump' or itemType == 'upsell')
if orderID ~= '' then
  if not isBumpOrUpsell and hasValue(session.order_ids, orderID) then
    mergePII(session, pii)
    cleanEmptyArrays(session)
    local updated = cjson.encode(session)
    redis.call('SET', KEYS[1], updated, 'EX', ttl)
    return {0, 1, updated}
  end
  if isBumpOrUpsell and type(session.items) == 'table' then
    for i = 1, #session.items do
      local existing = session.items[i]
      if existing.order_id == orderID and existing.item_type == itemType and existing.product_title == item.product_title then
        mergePII(session, pii)
        cleanEmptyArrays(session)
        local updated = cjson.encode(session)
        redis.call('SET', KEYS[1], updated, 'EX', ttl)
        return {0, 1, updated}
      end
    end
  end
end

if not item.item_type or item.item_type == '' or item.item_type == 'main' then
  if itemType ~= '' and itemType ~= 'main' then
    item.item_type = itemType
  else
    item.item_type = 'upsell'
  end
end

session.total_value = (tonumber(session.total_value) or 0) + val
session.count = (tonumber(session.count) or 0) + 1
if type(session.items) ~= 'table' then session.items = {} end
table.insert(session.items, item)

if orderID ~= '' then
  if type(session.order_ids) ~= 'table' then session.order_ids = {} end
  if not hasValue(session.order_ids, orderID) then table.insert(session.order_ids, orderID) end
end
if contentID ~= '' then
  if type(session.content_ids) ~= 'table' then session.content_ids = {} end
  if not hasValue(session.content_ids, contentID) then table.insert(session.content_ids, contentID) end
end
if contentName ~= '' then
  if type(session.content_names) ~= 'table' then session.content_names = {} end
  if not hasValue(session.content_names, contentName) then table.insert(session.content_names, contentName) end
end

mergePII(session, pii)
cleanEmptyArrays(session)
local encoded = cjson.encode(session)
redis.call('SET', KEYS[1], encoded, 'EX', ttl)
return {0, 0, encoded}
`)

type piiPatch struct {
	ExternalID      string `json:"external_id"`
	SourceURL       string `json:"source_url"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Country         string `json:"country"`
	State           string `json:"state"`
	City            string `json:"city"`
	PostalCode      string `json:"postal_code"`
	FBP             string `json:"fbp"`
	FBC             string `json:"fbc"`
	GCLID           string `json:"gclid"`
	WBraid          string `json:"wbraid"`
	GBraid          string `json:"gbraid"`
	TTClid          string `json:"ttclid"`
	TTP             string `json:"ttp"`
	GA4ClientID     string `json:"ga4_client_id"`
	GA4SessionID    string `json:"ga4_session_id"`
	ClientIP        string `json:"client_ip"`
	ClientUserAgent string `json:"client_user_agent"`
}

func evalAggregatePurchaseLua(
	ctx context.Context,
	rdb *redis.Client,
	key, pixelName string,
	in domain.SendEventInput,
	val float64,
	currentItem domain.AggregatedItem,
	ttl time.Duration,
) (isFirst bool, ignoredDup bool, session AggregatedSession, err error) {
	initial := newAggregatedSession(pixelName, in, val, currentItem)
	initBytes, err := json.Marshal(initial)
	if err != nil {
		return false, false, session, err
	}
	itemBytes, err := json.Marshal(currentItem)
	if err != nil {
		return false, false, session, err
	}
	piiBytes, err := json.Marshal(piiPatch{
		ExternalID: in.ExternalID, SourceURL: in.SourceURL,
		Email: in.Email, Phone: in.Phone, FirstName: in.FirstName, LastName: in.LastName,
		Country: in.Country, State: in.State, City: in.City, PostalCode: in.PostalCode,
		FBP: in.FBP, FBC: in.FBC, GCLID: in.GCLID, WBraid: in.WBraid, GBraid: in.GBraid,
		TTClid: in.TTClid, TTP: in.TTP, GA4ClientID: in.GA4ClientID, GA4SessionID: in.GA4SessionID,
		ClientIP: in.ClientIP, ClientUserAgent: in.ClientUserAgent,
	})
	if err != nil {
		return false, false, session, err
	}

	ttlSec := int(ttl.Seconds())
	if ttlSec < 60 {
		ttlSec = 60
	}
	res, err := aggregatePurchaseScript.Run(ctx, rdb, []string{key},
		string(initBytes),
		string(itemBytes),
		string(piiBytes),
		in.OrderID,
		in.ItemType,
		in.ContentID,
		in.ContentName,
		strconv.FormatFloat(val, 'f', -1, 64),
		ttlSec,
	).Slice()
	if err != nil {
		return false, false, session, err
	}
	if len(res) >= 3 {
		if v, ok := res[0].(int64); ok && v == 1 {
			isFirst = true
		}
		if v, ok := res[1].(int64); ok && v == 1 {
			ignoredDup = true
		}
		if rawStr, ok := res[2].(string); ok && rawStr != "" {
			_ = json.Unmarshal([]byte(rawStr), &session)
		}
	}
	return isFirst, ignoredDup, session, nil
}

