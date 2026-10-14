package service

import (
	"context"

	"marketplace/commerce-service/internal/domain"
	"marketplace/commerce-service/internal/repository"
)

func (s *Service) CreatePromoCode(ctx context.Context, principal domain.Principal, input domain.PromoCodeInput) (domain.PromoCode, error) {
	if err := requireRole(principal, domain.RoleSeller, domain.RoleAdmin); err != nil {
		return domain.PromoCode{}, err
	}
	if !input.ValidUntil.After(input.ValidFrom) {
		return domain.PromoCode{}, validationError("valid_until must be later than valid_from", map[string]any{
			"valid_until": "must be later than valid_from",
		})
	}

	promo, err := s.repository.CreatePromoCode(ctx, input)
	if repository.IsUniqueViolation(err) {
		return domain.PromoCode{}, newError(
			CodePromoAlreadyExists,
			"Promo code already exists",
			nil,
		)
	}
	if err != nil {
		return domain.PromoCode{}, internalError(err)
	}
	return promo, nil
}
