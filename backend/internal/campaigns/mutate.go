package campaigns

import (
	"context"
	"errors"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func formatGoogleAdsCampaignWarn(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if strings.Contains(strings.ToLower(msg), "unknown action") {
		return "Google Ads: sincronização de campanhas temporariamente indisponível. Tente novamente em instantes."
	}
	return "Google Ads: " + msg
}

func wrapGoogleAdsMutateErr(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "unknown action") {
		return errors.New("ação temporariamente indisponível na integração Google Ads; tente novamente em instantes")
	}
	return err
}

func (s *Service) MutateStatus(ctx context.Context, accountID int64, in domain.CampaignStatusMutateInput) error {
	switch strings.ToLower(strings.TrimSpace(in.Platform)) {
	case "meta":
		acc, permTok, err := s.resolveMetaCredsForMutation(ctx, accountID, in.AccountRefID, in.PixelName)
		if err != nil {
			return err
		}
		if err := s.metaOAuth.MutateObjectStatus(ctx, acc, permTok, in.ObjectID, in.Status); err != nil {
			return err
		}
	case "google":
		acc, err := s.resolveGoogleCredsForMutation(ctx, accountID, in.AccountRefID, in.PixelName)
		if err != nil {
			return err
		}
		if err := s.googleAds.MutateStatus(ctx, acc, in.AdAccountID, in.LoginCustomerID, in.Level, in.EntitySubtype, in.ObjectID, in.ParentID, in.Status); err != nil {
			return wrapGoogleAdsMutateErr(err)
		}
	default:
		return errors.New("plataforma inválida")
	}
	s.InvalidateAccountCache(ctx, accountID)
	return nil
}

func (s *Service) MutateBudget(ctx context.Context, accountID int64, in domain.CampaignBudgetMutateInput) error {
	if in.DailyBudget <= 0 {
		return errors.New("orçamento diário deve ser maior que zero")
	}
	switch strings.ToLower(strings.TrimSpace(in.Platform)) {
	case "meta":
		acc, permTok, err := s.resolveMetaCredsForMutation(ctx, accountID, in.AccountRefID, in.PixelName)
		if err != nil {
			return err
		}
		if err := s.metaOAuth.MutateObjectDailyBudget(ctx, acc, permTok, in.ObjectID, in.DailyBudget, in.Currency); err != nil {
			return err
		}
	case "google":
		acc, err := s.resolveGoogleCredsForMutation(ctx, accountID, in.AccountRefID, in.PixelName)
		if err != nil {
			return err
		}
		if err := s.googleAds.MutateBudget(ctx, acc, in.AdAccountID, in.LoginCustomerID, in.BudgetID, in.DailyBudget); err != nil {
			return wrapGoogleAdsMutateErr(err)
		}
	default:
		return errors.New("plataforma inválida")
	}
	s.InvalidateAccountCache(ctx, accountID)
	return nil
}
