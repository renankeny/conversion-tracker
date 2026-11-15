package domain

import (
	"strings"
	"time"
)

type GoogleAdsConversionAction struct {
	EventType          string `json:"event_type"`
	ConversionActionID string `json:"conversion_action_id"`
	Name               string `json:"name"`
	Enabled            bool   `json:"enabled"`
}

type GoogleAdsAccount struct {
	ID                 int64                       `json:"id"`
	AccountID          int64                       `json:"account_id"`
	Name               string                      `json:"name"`
	CustomerID         string                      `json:"customer_id"`
	CustomerName       string                      `json:"customer_name"`
	LoginCustomerID    string                      `json:"login_customer_id"`
	ConversionActionID string                      `json:"-"`
	ConversionActions  []GoogleAdsConversionAction `json:"conversion_actions"`
	RefreshToken       string                      `json:"-"`
	OAuthEmail         string                      `json:"oauth_email"`
	OAuthPictureURL    string                      `json:"oauth_picture_url"`
	Enabled            bool                        `json:"enabled"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
}

func (a GoogleAdsAccount) HasCredentials() bool {
	return strings.TrimSpace(a.RefreshToken) != "" && strings.TrimSpace(a.CustomerID) != ""
}

func (a GoogleAdsAccount) IsConnected() bool {
	if !a.HasCredentials() {
		return false
	}
	return a.HasActionForAnyEvent()
}

func (a GoogleAdsAccount) HasActionForAnyEvent() bool {
	for _, ca := range a.ConversionActions {
		if ca.Enabled && ca.ConversionActionID != "" && ca.EventType != "" {
			return true
		}
	}
	return false
}

func (a GoogleAdsAccount) FindActionsForEvent(eventType string) []*GoogleAdsConversionAction {
	out := make([]*GoogleAdsConversionAction, 0)
	isLeadAlias := eventType == EventLead || eventType == EventLeadSubmitted
	for i := range a.ConversionActions {
		ca := &a.ConversionActions[i]
		if !ca.Enabled || ca.ConversionActionID == "" {
			continue
		}
		if ca.EventType == eventType || (isLeadAlias && (ca.EventType == EventLead || ca.EventType == EventLeadSubmitted)) {
			out = append(out, ca)
		}
	}
	return out
}

func (a GoogleAdsAccount) FindActionForEvent(eventType string) *GoogleAdsConversionAction {
	actions := a.FindActionsForEvent(eventType)
	if len(actions) == 0 {
		return nil
	}
	return actions[0]
}

func (a GoogleAdsAccount) FindActionByIDOrEvent(actionID, eventType string) *GoogleAdsConversionAction {
	actionID = strings.TrimSpace(actionID)
	if actionID != "" {
		for i := range a.ConversionActions {
			ca := &a.ConversionActions[i]
			if strings.TrimSpace(ca.ConversionActionID) == actionID {
				return ca
			}
		}
		return &GoogleAdsConversionAction{
			EventType:          eventType,
			ConversionActionID: actionID,
			Enabled:            true,
		}
	}
	return a.FindActionForEvent(eventType)
}

func (a GoogleAdsAccount) PublicView() map[string]any {
	oauthOk := strings.TrimSpace(a.RefreshToken) != ""
	return map[string]any{
		"id":                 a.ID,
		"account_id":         a.AccountID,
		"name":               a.Name,
		"customer_id":        a.CustomerID,
		"customer_name":      a.CustomerName,
		"login_customer_id":  a.LoginCustomerID,
		"conversion_actions": a.ConversionActions,
		"oauth_email":        a.OAuthEmail,
		"oauth_picture_url":  a.OAuthPictureURL,
		"picture_url":        a.OAuthPictureURL,
		"oauth_connected":    oauthOk,
		"ready":              a.IsConnected(),
		"enabled":            a.Enabled,
		"connected":          a.IsConnected(),
		"created_at":         a.CreatedAt,
		"updated_at":         a.UpdatedAt,
	}
}
