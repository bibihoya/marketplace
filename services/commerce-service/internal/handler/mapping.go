package handler

import (
	"net/http"

	"marketplace/commerce-service/internal/api/generated"
	"marketplace/commerce-service/internal/domain"
)

func productCreateInput(request generated.ProductCreate) (domain.ProductInput, error) {
	price, ok := domain.Hundredths(request.Price)
	if !ok || price == 0 {
		return domain.ProductInput{}, invalidMoney("price")
	}

	input := domain.ProductInput{
		Name:        request.Name,
		Description: request.Description,
		PriceCents:  price,
		Stock:       request.Stock,
		Category:    request.Category,
		Status:      domain.ProductStatus(request.Status),
	}
	if request.SellerId != nil {
		input.SellerID = *request.SellerId
	}
	return input, nil
}

func productUpdateInput(request generated.ProductUpdate) (domain.ProductInput, error) {
	price, ok := domain.Hundredths(request.Price)
	if !ok || price == 0 {
		return domain.ProductInput{}, invalidMoney("price")
	}
	return domain.ProductInput{
		Name:        request.Name,
		Description: request.Description,
		PriceCents:  price,
		Stock:       request.Stock,
		Category:    request.Category,
		Status:      domain.ProductStatus(request.Status),
	}, nil
}

func promoCodeInput(request generated.PromoCodeCreate) (domain.PromoCodeInput, error) {
	discountValue, ok := domain.Hundredths(request.DiscountValue)
	if !ok || discountValue == 0 {
		return domain.PromoCodeInput{}, invalidMoney("discount_value")
	}
	minimum, ok := domain.Hundredths(request.MinOrderAmount)
	if !ok {
		return domain.PromoCodeInput{}, invalidMoney("min_order_amount")
	}

	active := true
	if request.Active != nil {
		active = *request.Active
	}
	return domain.PromoCodeInput{
		Code:                    request.Code,
		DiscountType:            domain.DiscountType(request.DiscountType),
		DiscountValueHundredths: discountValue,
		MinOrderAmountCents:     minimum,
		MaxUses:                 request.MaxUses,
		ValidFrom:               request.ValidFrom,
		ValidUntil:              request.ValidUntil,
		Active:                  active,
	}, nil
}

func orderItemInputs(request []generated.OrderItemRequest) []domain.OrderItemInput {
	items := make([]domain.OrderItemInput, 0, len(request))
	for _, item := range request {
		items = append(items, domain.OrderItemInput{
			ProductID: item.ProductId,
			Quantity:  item.Quantity,
		})
	}
	return items
}

func productResponse(product domain.Product) generated.ProductResponse {
	price := domain.Decimal(product.PriceCents)
	return generated.ProductResponse{
		Id:          &product.ID,
		SellerId:    &product.SellerID,
		Name:        product.Name,
		Description: product.Description,
		Price:       &price,
		Stock:       &product.Stock,
		Category:    product.Category,
		Status:      generated.ProductStatus(product.Status),
		CreatedAt:   &product.CreatedAt,
		UpdatedAt:   &product.UpdatedAt,
	}
}

func promoCodeResponse(promo domain.PromoCode) generated.PromoCodeResponse {
	discount := domain.Decimal(promo.DiscountValueHundredths)
	minimum := domain.Decimal(promo.MinOrderAmountCents)
	return generated.PromoCodeResponse{
		Id:             &promo.ID,
		Code:           &promo.Code,
		DiscountType:   generated.DiscountType(promo.DiscountType),
		DiscountValue:  &discount,
		MinOrderAmount: &minimum,
		MaxUses:        &promo.MaxUses,
		CurrentUses:    &promo.CurrentUses,
		ValidFrom:      &promo.ValidFrom,
		ValidUntil:     &promo.ValidUntil,
		Active:         &promo.Active,
	}
}

func orderResponse(order domain.Order) generated.OrderResponse {
	items := make([]generated.OrderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		price := domain.Decimal(item.PriceAtOrderCents)
		items = append(items, generated.OrderItemResponse{
			Id:           &item.ID,
			ProductId:    &item.ProductID,
			Quantity:     &item.Quantity,
			PriceAtOrder: &price,
		})
	}
	total := domain.Decimal(order.TotalAmountCents)
	discount := domain.Decimal(order.DiscountAmountCents)
	return generated.OrderResponse{
		Id:             &order.ID,
		UserId:         &order.UserID,
		Status:         generated.OrderStatus(order.Status),
		Items:          items,
		PromoCode:      order.PromoCode,
		TotalAmount:    &total,
		DiscountAmount: &discount,
		CreatedAt:      &order.CreatedAt,
		UpdatedAt:      &order.UpdatedAt,
	}
}

func invalidMoney(field string) error {
	return newHTTPError(http.StatusBadRequest, generated.VALIDATIONERROR, "Invalid monetary value", map[string]any{
		field: "must be a decimal value with at most two fractional digits",
	})
}
