package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"marketplace/commerce-service/internal/api/generated"
	"marketplace/commerce-service/internal/auth"
	"marketplace/commerce-service/internal/domain"
)

type CommerceService interface {
	ListProducts(context.Context, domain.Principal, domain.ProductFilter) ([]domain.Product, int64, error)
	GetProduct(context.Context, domain.Principal, uuid.UUID) (domain.Product, error)
	CreateProduct(context.Context, domain.Principal, domain.ProductInput) (domain.Product, error)
	UpdateProduct(context.Context, domain.Principal, uuid.UUID, domain.ProductInput) (domain.Product, error)
	ArchiveProduct(context.Context, domain.Principal, uuid.UUID) (domain.Product, error)
	CreatePromoCode(context.Context, domain.Principal, domain.PromoCodeInput) (domain.PromoCode, error)
	CreateOrder(context.Context, domain.Principal, []domain.OrderItemInput, *string) (domain.Order, error)
	GetOrder(context.Context, domain.Principal, uuid.UUID) (domain.Order, error)
	UpdateOrder(context.Context, domain.Principal, uuid.UUID, []domain.OrderItemInput) (domain.Order, error)
	CancelOrder(context.Context, domain.Principal, uuid.UUID) (domain.Order, error)
	ChangeOrderStatus(context.Context, domain.Principal, uuid.UUID, domain.OrderStatus) (domain.Order, error)
}

type Server struct {
	service CommerceService
	logger  *slog.Logger
}

var _ generated.StrictServerInterface = (*Server)(nil)

func (s *Server) GetHealth(_ context.Context, _ generated.GetHealthRequestObject) (generated.GetHealthResponseObject, error) {
	return generated.GetHealth200JSONResponse{
		Status:  generated.Ok,
		Service: "commerce-service",
	}, nil
}

func (s *Server) ListProducts(ctx context.Context, request generated.ListProductsRequestObject) (generated.ListProductsResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}

	page := int32(0)
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	size := int32(20)
	if request.Params.Size != nil {
		size = *request.Params.Size
	}
	var status *domain.ProductStatus
	if request.Params.Status != nil {
		value := domain.ProductStatus(*request.Params.Status)
		status = &value
	}

	products, total, err := s.service.ListProducts(ctx, principal, domain.ProductFilter{
		Page:     page,
		Size:     size,
		Status:   status,
		Category: request.Params.Category,
	})
	if err != nil {
		return nil, err
	}

	items := make([]generated.ProductResponse, 0, len(products))
	for _, product := range products {
		items = append(items, productResponse(product))
	}
	return generated.ListProducts200JSONResponse{
		Items:         items,
		TotalElements: total,
		Page:          page,
		Size:          size,
	}, nil
}

func (s *Server) CreateProduct(ctx context.Context, request generated.CreateProductRequestObject) (generated.CreateProductResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBody()
	}

	input, err := productCreateInput(*request.Body)
	if err != nil {
		return nil, err
	}
	product, err := s.service.CreateProduct(ctx, principal, input)
	if err != nil {
		return nil, err
	}
	return generated.CreateProduct201JSONResponse(productResponse(product)), nil
}

func (s *Server) GetProduct(ctx context.Context, request generated.GetProductRequestObject) (generated.GetProductResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	product, err := s.service.GetProduct(ctx, principal, request.Id)
	if err != nil {
		return nil, err
	}
	return generated.GetProduct200JSONResponse(productResponse(product)), nil
}

func (s *Server) UpdateProduct(ctx context.Context, request generated.UpdateProductRequestObject) (generated.UpdateProductResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBody()
	}

	input, err := productUpdateInput(*request.Body)
	if err != nil {
		return nil, err
	}
	product, err := s.service.UpdateProduct(ctx, principal, request.Id, input)
	if err != nil {
		return nil, err
	}
	return generated.UpdateProduct200JSONResponse(productResponse(product)), nil
}

func (s *Server) ArchiveProduct(ctx context.Context, request generated.ArchiveProductRequestObject) (generated.ArchiveProductResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	product, err := s.service.ArchiveProduct(ctx, principal, request.Id)
	if err != nil {
		return nil, err
	}
	return generated.ArchiveProduct200JSONResponse(productResponse(product)), nil
}

func (s *Server) CreatePromoCode(ctx context.Context, request generated.CreatePromoCodeRequestObject) (generated.CreatePromoCodeResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBody()
	}

	input, err := promoCodeInput(*request.Body)
	if err != nil {
		return nil, err
	}
	promo, err := s.service.CreatePromoCode(ctx, principal, input)
	if err != nil {
		return nil, err
	}
	return generated.CreatePromoCode201JSONResponse(promoCodeResponse(promo)), nil
}

func (s *Server) CreateOrder(ctx context.Context, request generated.CreateOrderRequestObject) (generated.CreateOrderResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBody()
	}

	order, err := s.service.CreateOrder(ctx, principal, orderItemInputs(request.Body.Items), request.Body.PromoCode)
	if err != nil {
		return nil, err
	}
	return generated.CreateOrder201JSONResponse(orderResponse(order)), nil
}

func (s *Server) GetOrder(ctx context.Context, request generated.GetOrderRequestObject) (generated.GetOrderResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	order, err := s.service.GetOrder(ctx, principal, request.Id)
	if err != nil {
		return nil, err
	}
	return generated.GetOrder200JSONResponse(orderResponse(order)), nil
}

func (s *Server) UpdateOrder(ctx context.Context, request generated.UpdateOrderRequestObject) (generated.UpdateOrderResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBody()
	}
	order, err := s.service.UpdateOrder(ctx, principal, request.Id, orderItemInputs(request.Body.Items))
	if err != nil {
		return nil, err
	}
	return generated.UpdateOrder200JSONResponse(orderResponse(order)), nil
}

func (s *Server) CancelOrder(ctx context.Context, request generated.CancelOrderRequestObject) (generated.CancelOrderResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	order, err := s.service.CancelOrder(ctx, principal, request.Id)
	if err != nil {
		return nil, err
	}
	return generated.CancelOrder200JSONResponse(orderResponse(order)), nil
}

func (s *Server) ChangeOrderStatus(ctx context.Context, request generated.ChangeOrderStatusRequestObject) (generated.ChangeOrderStatusResponseObject, error) {
	principal, err := authenticatedPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBody()
	}
	order, err := s.service.ChangeOrderStatus(ctx, principal, request.Id, domain.OrderStatus(request.Body.Status))
	if err != nil {
		return nil, err
	}
	return generated.ChangeOrderStatus200JSONResponse(orderResponse(order)), nil
}

func authenticatedPrincipal(ctx context.Context) (domain.Principal, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return domain.Principal{}, auth.NewError(auth.CodeTokenInvalid, "Access token is invalid")
	}
	return principal, nil
}

func missingBody() error {
	return newHTTPError(http.StatusBadRequest, generated.VALIDATIONERROR, "Request body is required", map[string]any{
		"body": "is required",
	})
}
