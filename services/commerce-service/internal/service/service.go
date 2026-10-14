package service

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"marketplace/commerce-service/internal/domain"
	"marketplace/commerce-service/internal/repository"
)

type Service struct {
	repository repository.Repository
	rateLimit  time.Duration
	now        func() time.Time
}

func New(repository repository.Repository, rateLimit time.Duration) *Service {
	return &Service{
		repository: repository,
		rateLimit:  rateLimit,
		now:        time.Now,
	}
}

func requireRole(principal domain.Principal, roles ...domain.Role) error {
	for _, role := range roles {
		if principal.Role == role {
			return nil
		}
	}

	return newError(CodeAccessDenied, "Access denied", nil)
}

func ensureOrderAccess(principal domain.Principal, ownerID uuid.UUID) error {
	if principal.Role == domain.RoleAdmin {
		return nil
	}
	if principal.UserID != ownerID {
		return newError(
			CodeOrderOwnershipViolation,
			"Order belongs to another user",
			nil,
		)
	}
	return nil
}

func sortedProductIDs(items []domain.OrderItemInput) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{}, len(items))
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if _, exists := seen[item.ProductID]; exists {
			return nil, validationError("Product IDs in an order must be unique", map[string]any{
				"product_id": item.ProductID.String(),
			})
		}
		seen[item.ProductID] = struct{}{}
		ids = append(ids, item.ProductID)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids, nil
}

func sortedCombinedProductIDs(oldItems []domain.OrderItem, newItems []domain.OrderItemInput) ([]uuid.UUID, error) {
	newIDs, err := sortedProductIDs(newItems)
	if err != nil {
		return nil, err
	}

	seen := make(map[uuid.UUID]struct{}, len(oldItems)+len(newIDs))
	ids := make([]uuid.UUID, 0, len(oldItems)+len(newIDs))
	for _, item := range oldItems {
		if _, exists := seen[item.ProductID]; !exists {
			seen[item.ProductID] = struct{}{}
			ids = append(ids, item.ProductID)
		}
	}
	for _, id := range newIDs {
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids, nil
}

func calculateSubtotal(items []domain.OrderItemInput, products map[uuid.UUID]domain.Product) (int64, []domain.OrderItem, error) {
	var subtotal int64
	orderItems := make([]domain.OrderItem, 0, len(items))

	for _, item := range items {
		if item.Quantity < 1 || item.Quantity > 999 {
			return 0, nil, validationError("Order item quantity is invalid", map[string]any{
				"product_id": item.ProductID.String(),
				"quantity":   "must be between 1 and 999",
			})
		}
		product := products[item.ProductID]
		quantity := int64(item.Quantity)
		if product.PriceCents > math.MaxInt64/quantity {
			return 0, nil, validationError("Order amount is too large", nil)
		}

		lineAmount := product.PriceCents * quantity
		if subtotal > domain.MaxNumericCents-lineAmount {
			return 0, nil, validationError("Order amount is too large", nil)
		}
		subtotal += lineAmount

		orderItems = append(orderItems, domain.OrderItem{
			ProductID:         item.ProductID,
			Quantity:          item.Quantity,
			PriceAtOrderCents: product.PriceCents,
		})
	}

	return subtotal, orderItems, nil
}

func calculateDiscount(subtotal int64, promo domain.PromoCode) int64 {
	if promo.DiscountType == domain.DiscountTypeFixedAmount {
		return min(promo.DiscountValueHundredths, subtotal)
	}

	percentage := promo.DiscountValueHundredths
	if percentage > 7000 {
		percentage = 7000
	}
	return (subtotal*percentage + 5000) / 10000
}

func (s *Service) checkRateLimit(ctx context.Context, tx repository.Transaction, userID uuid.UUID, operation domain.UserOperationType, now time.Time) error {
	lastOperation, err := tx.LastUserOperation(ctx, userID, operation)
	if err != nil {
		return internalError(err)
	}
	if lastOperation == nil || s.rateLimit == 0 {
		return nil
	}

	retryAfter := lastOperation.Add(s.rateLimit).Sub(now)
	if retryAfter <= 0 {
		return nil
	}

	return newError(
		CodeOrderLimitExceeded,
		"Order operation rate limit exceeded",
		map[string]any{"retry_after_seconds": int64(math.Ceil(retryAfter.Seconds()))},
	)
}

func promoIsActive(promo domain.PromoCode, now time.Time) bool {
	return promo.Active &&
		promo.CurrentUses < promo.MaxUses &&
		!now.Before(promo.ValidFrom) &&
		!now.After(promo.ValidUntil)
}
