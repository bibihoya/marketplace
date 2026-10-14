package service

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"marketplace/commerce-service/internal/domain"
	"marketplace/commerce-service/internal/repository"
)

func (s *Service) CreateOrder(ctx context.Context, principal domain.Principal, items []domain.OrderItemInput, promoCode *string) (domain.Order, error) {
	if err := requireRole(principal, domain.RoleUser, domain.RoleAdmin); err != nil {
		return domain.Order{}, err
	}

	productIDs, err := sortedProductIDs(items)
	if err != nil {
		return domain.Order{}, err
	}

	var result domain.Order
	err = s.repository.WithTx(ctx, func(tx repository.Transaction) error {
		now := s.now()
		if err := tx.LockUser(ctx, principal.UserID); err != nil {
			return internalError(err)
		}
		if err := s.checkRateLimit(ctx, tx, principal.UserID, domain.UserOperationCreateOrder, now); err != nil {
			return err
		}

		hasActiveOrder, err := tx.HasActiveOrder(ctx, principal.UserID)
		if err != nil {
			return internalError(err)
		}
		if hasActiveOrder {
			return newError(CodeOrderHasActive, "User already has an active order", nil)
		}

		products, err := tx.LockProducts(ctx, productIDs)
		if err != nil {
			return internalError(err)
		}
		if err := validateProducts(items, products); err != nil {
			return err
		}

		subtotal, orderItems, err := calculateSubtotal(items, products)
		if err != nil {
			return err
		}

		var promoID *uuid.UUID
		var appliedPromoCode *string
		var discount int64
		if promoCode != nil {
			promo, err := tx.LockPromoCodeByCode(ctx, *promoCode)
			if repository.IsNotFound(err) {
				return invalidPromoCode()
			}
			if err != nil {
				return internalError(err)
			}
			if !promoIsActive(promo, now) {
				return invalidPromoCode()
			}
			if subtotal < promo.MinOrderAmountCents {
				return promoMinAmount(promo.MinOrderAmountCents, subtotal)
			}

			discount = calculateDiscount(subtotal, promo)
			promoID = &promo.ID
			appliedPromoCode = &promo.Code
		}

		for _, item := range items {
			product := products[item.ProductID]
			if err := tx.SetProductStock(ctx, item.ProductID, product.Stock-item.Quantity); err != nil {
				return internalError(err)
			}
		}

		created, err := tx.CreateOrder(ctx, domain.Order{
			UserID:              principal.UserID,
			Status:              domain.OrderStatusCreated,
			PromoCodeID:         promoID,
			PromoCode:           appliedPromoCode,
			TotalAmountCents:    subtotal - discount,
			DiscountAmountCents: discount,
		})
		if repository.IsUniqueViolation(err) {
			return newError(CodeOrderHasActive, "User already has an active order", nil)
		}
		if err != nil {
			return internalError(err)
		}

		if promoID != nil {
			if err := tx.ChangePromoCodeUses(ctx, *promoID, 1); err != nil {
				return internalError(err)
			}
		}
		createdItems, err := tx.CreateOrderItems(ctx, created.ID, orderItems)
		if err != nil {
			return internalError(err)
		}
		if err := tx.RecordUserOperation(ctx, principal.UserID, domain.UserOperationCreateOrder); err != nil {
			return internalError(err)
		}

		created.Items = createdItems
		result = created
		return nil
	})
	if err != nil {
		return domain.Order{}, internalError(err)
	}

	return result, nil
}

func (s *Service) GetOrder(ctx context.Context, principal domain.Principal, id uuid.UUID) (domain.Order, error) {
	if err := requireRole(principal, domain.RoleUser, domain.RoleAdmin); err != nil {
		return domain.Order{}, err
	}

	order, err := s.repository.GetOrder(ctx, id)
	if repository.IsNotFound(err) {
		return domain.Order{}, orderNotFound()
	}
	if err != nil {
		return domain.Order{}, internalError(err)
	}
	if err := ensureOrderAccess(principal, order.UserID); err != nil {
		return domain.Order{}, err
	}

	items, err := s.repository.ListOrderItems(ctx, id)
	if err != nil {
		return domain.Order{}, internalError(err)
	}
	order.Items = items
	return order, nil
}

func (s *Service) UpdateOrder(ctx context.Context, principal domain.Principal, id uuid.UUID, newItems []domain.OrderItemInput) (domain.Order, error) {
	if err := requireRole(principal, domain.RoleUser, domain.RoleAdmin); err != nil {
		return domain.Order{}, err
	}
	if _, err := sortedProductIDs(newItems); err != nil {
		return domain.Order{}, err
	}

	var result domain.Order
	err := s.repository.WithTx(ctx, func(tx repository.Transaction) error {
		ownerID, err := tx.FindOrderOwner(ctx, id)
		if repository.IsNotFound(err) {
			return orderNotFound()
		}
		if err != nil {
			return internalError(err)
		}
		if err := ensureOrderAccess(principal, ownerID); err != nil {
			return err
		}

		if err := tx.LockUser(ctx, ownerID); err != nil {
			return internalError(err)
		}
		order, err := tx.LockOrder(ctx, id)
		if repository.IsNotFound(err) {
			return orderNotFound()
		}
		if err != nil {
			return internalError(err)
		}
		if order.Status != domain.OrderStatusCreated {
			return invalidStateTransition(order.Status, domain.OrderStatusCreated)
		}

		now := s.now()
		if err := s.checkRateLimit(ctx, tx, ownerID, domain.UserOperationUpdateOrder, now); err != nil {
			return err
		}
		oldItems, err := tx.ListOrderItems(ctx, id)
		if err != nil {
			return internalError(err)
		}
		productIDs, err := sortedCombinedProductIDs(oldItems, newItems)
		if err != nil {
			return err
		}
		products, err := tx.LockProducts(ctx, productIDs)
		if err != nil {
			return internalError(err)
		}

		availableStock := make(map[uuid.UUID]int32, len(products))
		for id, product := range products {
			availableStock[id] = product.Stock
		}
		for _, item := range oldItems {
			availableStock[item.ProductID] += item.Quantity
		}
		if err := validateProductsWithStock(newItems, products, availableStock); err != nil {
			return err
		}

		subtotal, replacementItems, err := calculateSubtotal(newItems, products)
		if err != nil {
			return err
		}

		promoID := order.PromoCodeID
		promoCode := order.PromoCode
		var discount int64
		if promoID != nil {
			promo, err := tx.LockPromoCodeByID(ctx, *promoID)
			if err != nil && !repository.IsNotFound(err) {
				return internalError(err)
			}
			stillApplicable := err == nil && promo.Active && !now.Before(promo.ValidFrom) && !now.After(promo.ValidUntil) && subtotal >= promo.MinOrderAmountCents
			if stillApplicable {
				discount = calculateDiscount(subtotal, promo)
			} else {
				if err == nil {
					if err := tx.ChangePromoCodeUses(ctx, promo.ID, -1); err != nil {
						return internalError(err)
					}
				}
				promoID = nil
				promoCode = nil
			}
		}

		newQuantities := make(map[uuid.UUID]int32, len(newItems))
		for _, item := range newItems {
			newQuantities[item.ProductID] = item.Quantity
		}
		for _, productID := range productIDs {
			stock := availableStock[productID] - newQuantities[productID]
			if err := tx.SetProductStock(ctx, productID, stock); err != nil {
				return internalError(err)
			}
		}

		if err := tx.DeleteOrderItems(ctx, id); err != nil {
			return internalError(err)
		}
		createdItems, err := tx.CreateOrderItems(ctx, id, replacementItems)
		if err != nil {
			return internalError(err)
		}
		order.PromoCodeID = promoID
		order.PromoCode = promoCode
		order.TotalAmountCents = subtotal - discount
		order.DiscountAmountCents = discount
		updated, err := tx.UpdateOrder(ctx, order)
		if err != nil {
			return internalError(err)
		}
		if err := tx.RecordUserOperation(ctx, ownerID, domain.UserOperationUpdateOrder); err != nil {
			return internalError(err)
		}

		updated.Items = createdItems
		result = updated
		return nil
	})
	if err != nil {
		return domain.Order{}, internalError(err)
	}
	return result, nil
}

func (s *Service) CancelOrder(ctx context.Context, principal domain.Principal, id uuid.UUID) (domain.Order, error) {
	if err := requireRole(principal, domain.RoleUser, domain.RoleAdmin); err != nil {
		return domain.Order{}, err
	}

	var result domain.Order
	err := s.repository.WithTx(ctx, func(tx repository.Transaction) error {
		ownerID, err := tx.FindOrderOwner(ctx, id)
		if repository.IsNotFound(err) {
			return orderNotFound()
		}
		if err != nil {
			return internalError(err)
		}
		if err := ensureOrderAccess(principal, ownerID); err != nil {
			return err
		}

		if err := tx.LockUser(ctx, ownerID); err != nil {
			return internalError(err)
		}
		order, err := tx.LockOrder(ctx, id)
		if repository.IsNotFound(err) {
			return orderNotFound()
		}
		if err != nil {
			return internalError(err)
		}
		if order.Status != domain.OrderStatusCreated && order.Status != domain.OrderStatusPaymentPending {
			return invalidStateTransition(order.Status, domain.OrderStatusCanceled)
		}

		items, err := tx.ListOrderItems(ctx, id)
		if err != nil {
			return internalError(err)
		}
		productIDs := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
			productIDs = append(productIDs, item.ProductID)
		}
		sort.Slice(productIDs, func(i, j int) bool { return productIDs[i].String() < productIDs[j].String() })
		products, err := tx.LockProducts(ctx, productIDs)
		if err != nil {
			return internalError(err)
		}
		for _, item := range items {
			product, exists := products[item.ProductID]
			if !exists {
				return internalError(productNotFound())
			}
			if err := tx.SetProductStock(ctx, item.ProductID, product.Stock+item.Quantity); err != nil {
				return internalError(err)
			}
		}

		if order.PromoCodeID != nil {
			promo, err := tx.LockPromoCodeByID(ctx, *order.PromoCodeID)
			if err != nil {
				return internalError(err)
			}
			if err := tx.ChangePromoCodeUses(ctx, promo.ID, -1); err != nil {
				return internalError(err)
			}
		}

		canceled, err := tx.SetOrderStatus(ctx, id, domain.OrderStatusCanceled)
		if err != nil {
			return internalError(err)
		}
		canceled.Items = items
		result = canceled
		return nil
	})
	if err != nil {
		return domain.Order{}, internalError(err)
	}
	return result, nil
}

func (s *Service) ChangeOrderStatus(ctx context.Context, principal domain.Principal, id uuid.UUID, status domain.OrderStatus) (domain.Order, error) {
	if err := requireRole(principal, domain.RoleAdmin); err != nil {
		return domain.Order{}, err
	}

	var result domain.Order
	err := s.repository.WithTx(ctx, func(tx repository.Transaction) error {
		order, err := tx.LockOrder(ctx, id)
		if repository.IsNotFound(err) {
			return orderNotFound()
		}
		if err != nil {
			return internalError(err)
		}

		expectedNext := map[domain.OrderStatus]domain.OrderStatus{
			domain.OrderStatusCreated:        domain.OrderStatusPaymentPending,
			domain.OrderStatusPaymentPending: domain.OrderStatusPaid,
			domain.OrderStatusPaid:           domain.OrderStatusShipped,
			domain.OrderStatusShipped:        domain.OrderStatusCompleted,
		}
		if expectedNext[order.Status] != status {
			return invalidStateTransition(order.Status, status)
		}

		updated, err := tx.SetOrderStatus(ctx, id, status)
		if err != nil {
			return internalError(err)
		}
		items, err := tx.ListOrderItems(ctx, id)
		if err != nil {
			return internalError(err)
		}
		updated.Items = items
		result = updated
		return nil
	})
	if err != nil {
		return domain.Order{}, internalError(err)
	}
	return result, nil
}

func validateProducts(items []domain.OrderItemInput, products map[uuid.UUID]domain.Product) error {
	available := make(map[uuid.UUID]int32, len(products))
	for id, product := range products {
		available[id] = product.Stock
	}
	return validateProductsWithStock(items, products, available)
}

func validateProductsWithStock(items []domain.OrderItemInput, products map[uuid.UUID]domain.Product, available map[uuid.UUID]int32) error {
	for _, item := range items {
		if _, exists := products[item.ProductID]; !exists {
			return newError(CodeProductNotFound, "Product not found", map[string]any{
				"product_id": item.ProductID.String(),
			})
		}
	}
	for _, item := range items {
		if products[item.ProductID].Status != domain.ProductStatusActive {
			return newError(CodeProductInactive, "Product is not active", map[string]any{
				"product_id": item.ProductID.String(),
			})
		}
	}

	insufficient := make([]map[string]any, 0)
	for _, item := range items {
		stock := available[item.ProductID]
		if stock < item.Quantity {
			insufficient = append(insufficient, map[string]any{
				"product_id":         item.ProductID.String(),
				"requested_quantity": item.Quantity,
				"available_stock":    stock,
			})
		}
	}
	if len(insufficient) > 0 {
		return newError(CodeInsufficientStock, "Insufficient product stock", map[string]any{
			"products": insufficient,
		})
	}
	return nil
}

func invalidPromoCode() error {
	return newError(CodePromoInvalid, "Promo code is invalid", nil)
}

func promoMinAmount(minimum, actual int64) error {
	return newError(CodePromoMinAmount, "Order amount is below promo code minimum", map[string]any{
		"minimum_amount": domain.Decimal(minimum),
		"actual_amount":  domain.Decimal(actual),
	})
}

func invalidStateTransition(from, to domain.OrderStatus) error {
	return newError(CodeInvalidStateTransition, "Invalid order state transition", map[string]any{
		"from": from,
		"to":   to,
	})
}
