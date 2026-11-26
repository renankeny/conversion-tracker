package campaigns

import (
	"context"
	"errors"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type resolvedMetaTarget struct {
	acc         *domain.MetaAccount
	adAccountID string
	permToken   string
}

type resolvedGoogleTarget struct {
	acc        *domain.GoogleAdsAccount
	customerID string
	loginCID   string
}

func (s *Service) resolveTargets(ctx context.Context, p OverviewParams) (domain.PixelFinancialConfig, resolvedMetaTarget, resolvedGoogleTarget) {
	var fin domain.PixelFinancialConfig
	var mt resolvedMetaTarget
	var gt resolvedGoogleTarget

	var pixel *domain.Pixel
	if p.PixelName != "" && s.pixels != nil {
		if px, err := s.pixels.FindByName(ctx, p.PixelName); err == nil && px != nil && (p.AccountID == 0 || px.AccountID == p.AccountID) {
			pixel = px
			fin = domain.PixelFinancialConfig{
				GatewayFeePct:   px.GatewayFeePct,
				GatewayFixedFee: px.GatewayFixedFee,
				TaxPct:          px.TaxPct,
				ProductCost:     px.ProductCost,
			}
		}
	}

	// Meta target resolution: seleção explícita -> vínculo do Pixel -> primeira conta Meta conectada da empresa
	if p.MetaAccountID > 0 && s.metaAccs != nil {
		if acc, err := s.metaAccs.FindByID(ctx, p.MetaAccountID); err == nil && acc != nil && (p.AccountID == 0 || acc.AccountID == p.AccountID) {
			mt.acc = acc
			mt.adAccountID = firstNonEmpty(p.MetaAdAccountID, acc.AdAccountID)
		}
	} else if pixel != nil {
		if pixel.MetaAccountID != nil && *pixel.MetaAccountID > 0 && s.metaAccs != nil {
			if acc, err := s.metaAccs.FindByID(ctx, *pixel.MetaAccountID); err == nil && acc != nil {
				mt.acc = acc
				mt.adAccountID = firstNonEmpty(p.MetaAdAccountID, pixel.MetaAdAccountID, acc.AdAccountID)
			}
		} else if pixel.MetaPermanentToken != "" && pixel.MetaAdAccountID != "" {
			mt.permToken = pixel.MetaPermanentToken
			mt.adAccountID = firstNonEmpty(p.MetaAdAccountID, pixel.MetaAdAccountID)
		}
	}
	if mt.adAccountID == "" && s.metaAccs != nil && p.Platform != "google" {
		if list, err := s.metaAccs.ListByAccount(ctx, p.AccountID); err == nil {
			for i := range list {
				if list[i].Enabled && list[i].IsConnected() && list[i].AdAccountID != "" {
					accCopy := list[i]
					mt.acc = &accCopy
					mt.adAccountID = firstNonEmpty(p.MetaAdAccountID, accCopy.AdAccountID)
					break
				}
			}
		}
	}

	// Google Ads target resolution: seleção explícita -> vínculo do Pixel -> primeira conta Google conectada da empresa
	if p.GoogleAdsAccountID > 0 && s.gadsAccs != nil {
		if acc, err := s.gadsAccs.FindByID(ctx, p.GoogleAdsAccountID); err == nil && acc != nil && (p.AccountID == 0 || acc.AccountID == p.AccountID) {
			gt.acc = acc
			gt.customerID = firstNonEmpty(p.GoogleCustomerID, acc.CustomerID)
			gt.loginCID = acc.LoginCustomerID
		}
	} else if pixel != nil && pixel.HasGoogleAds() && s.gadsAccs != nil {
		if acc, err := s.gadsAccs.FindByID(ctx, *pixel.GoogleAdsAccountID); err == nil && acc != nil {
			gt.acc = acc
			gt.customerID = firstNonEmpty(p.GoogleCustomerID, pixel.GoogleAdsCustomerID, acc.CustomerID)
			gt.loginCID = firstNonEmpty(pixel.GoogleAdsLoginCustomerID, acc.LoginCustomerID)
		}
	}
	if gt.acc == nil && s.gadsAccs != nil && p.Platform != "meta" {
		if list, err := s.gadsAccs.ListByAccount(ctx, p.AccountID); err == nil {
			for i := range list {
				if list[i].Enabled && list[i].HasCredentials() {
					accCopy := list[i]
					gt.acc = &accCopy
					gt.customerID = firstNonEmpty(p.GoogleCustomerID, accCopy.CustomerID)
					gt.loginCID = accCopy.LoginCustomerID
					break
				}
			}
		}
	}

	return fin, mt, gt
}

func (s *Service) resolveMetaCredsForMutation(ctx context.Context, accountID, refID int64, pixelName string) (*domain.MetaAccount, string, error) {
	if refID > 0 && s.metaAccs != nil {
		acc, err := s.metaAccs.FindByID(ctx, refID)
		if err == nil && acc != nil && (accountID == 0 || acc.AccountID == accountID) {
			return acc, "", nil
		}
	}
	if pixelName != "" && s.pixels != nil {
		if px, err := s.pixels.FindByName(ctx, pixelName); err == nil && px != nil && (accountID == 0 || px.AccountID == accountID) {
			if px.MetaAccountID != nil && *px.MetaAccountID > 0 && s.metaAccs != nil {
				if acc, err := s.metaAccs.FindByID(ctx, *px.MetaAccountID); err == nil && acc != nil {
					return acc, "", nil
				}
			}
			if strings.TrimSpace(px.MetaPermanentToken) != "" {
				return nil, px.MetaPermanentToken, nil
			}
		}
	}
	return nil, "", errors.New("conta Meta Ads não autorizada ou não encontrada para esta ação")
}

func (s *Service) resolveGoogleCredsForMutation(ctx context.Context, accountID, refID int64, pixelName string) (*domain.GoogleAdsAccount, error) {
	if refID > 0 && s.gadsAccs != nil {
		acc, err := s.gadsAccs.FindByID(ctx, refID)
		if err == nil && acc != nil && (accountID == 0 || acc.AccountID == accountID) {
			return acc, nil
		}
	}
	if pixelName != "" && s.pixels != nil {
		if px, err := s.pixels.FindByName(ctx, pixelName); err == nil && px != nil && px.HasGoogleAds() && (accountID == 0 || px.AccountID == accountID) {
			if acc, err := s.gadsAccs.FindByID(ctx, *px.GoogleAdsAccountID); err == nil && acc != nil {
				return acc, nil
			}
		}
	}
	return nil, errors.New("conta Google Ads não autorizada ou não encontrada para esta ação")
}
