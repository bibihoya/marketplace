package httpapi

import (
	"context"
	"errors"
	"net/http"

	"marketplace/commerce-service/internal/api/generated"
)

var (
	errNotImplemented                                 = errors.New("operation is not implemented")
	_                 generated.StrictServerInterface = (*Server)(nil)
)

// Server implements the generated strict API contract.
type Server struct{}

// NewHandler creates an HTTP handler from the generated OpenAPI routes.
func NewHandler() http.Handler {
	strictHandler := generated.NewStrictHandlerWithOptions(
		&Server{},
		nil,
		generated.StrictHTTPServerOptions{
			RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			},
			ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
				if errors.Is(err, errNotImplemented) {
					http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
					return
				}

				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			},
		},
	)

	return generated.Handler(strictHandler)
}

// GetHealth returns the current service status.
func (s *Server) GetHealth(_ context.Context, _ generated.GetHealthRequestObject) (generated.GetHealthResponseObject, error) {
	return generated.GetHealth200JSONResponse{
		Status:  generated.Ok,
		Service: "commerce-service",
	}, nil
}

// CreateOrder is generated from POST /orders and will be implemented in the orders stage.
func (s *Server) CreateOrder(context.Context, generated.CreateOrderRequestObject) (generated.CreateOrderResponseObject, error) {
	return nil, errNotImplemented
}

// GetOrder is generated from GET /orders/{id} and will be implemented in the orders stage.
func (s *Server) GetOrder(context.Context, generated.GetOrderRequestObject) (generated.GetOrderResponseObject, error) {
	return nil, errNotImplemented
}

// UpdateOrder is generated from PUT /orders/{id} and will be implemented in the orders stage.
func (s *Server) UpdateOrder(context.Context, generated.UpdateOrderRequestObject) (generated.UpdateOrderResponseObject, error) {
	return nil, errNotImplemented
}

// CancelOrder is generated from POST /orders/{id}/cancel and will be implemented in the orders stage.
func (s *Server) CancelOrder(context.Context, generated.CancelOrderRequestObject) (generated.CancelOrderResponseObject, error) {
	return nil, errNotImplemented
}

// ChangeOrderStatus is generated from PATCH /orders/{id}/status and will be implemented in the orders stage.
func (s *Server) ChangeOrderStatus(context.Context, generated.ChangeOrderStatusRequestObject) (generated.ChangeOrderStatusResponseObject, error) {
	return nil, errNotImplemented
}

// ListProducts is generated from GET /products and will be implemented in the product CRUD stage.
func (s *Server) ListProducts(context.Context, generated.ListProductsRequestObject) (generated.ListProductsResponseObject, error) {
	return nil, errNotImplemented
}

// CreateProduct is generated from POST /products and will be implemented in the product CRUD stage.
func (s *Server) CreateProduct(context.Context, generated.CreateProductRequestObject) (generated.CreateProductResponseObject, error) {
	return nil, errNotImplemented
}

// ArchiveProduct is generated from DELETE /products/{id} and will be implemented in the product CRUD stage.
func (s *Server) ArchiveProduct(context.Context, generated.ArchiveProductRequestObject) (generated.ArchiveProductResponseObject, error) {
	return nil, errNotImplemented
}

// GetProduct is generated from GET /products/{id} and will be implemented in the product CRUD stage.
func (s *Server) GetProduct(context.Context, generated.GetProductRequestObject) (generated.GetProductResponseObject, error) {
	return nil, errNotImplemented
}

// UpdateProduct is generated from PUT /products/{id} and will be implemented in the product CRUD stage.
func (s *Server) UpdateProduct(context.Context, generated.UpdateProductRequestObject) (generated.UpdateProductResponseObject, error) {
	return nil, errNotImplemented
}

// CreatePromoCode is generated from POST /promo-codes and will be implemented in the promotions stage.
func (s *Server) CreatePromoCode(context.Context, generated.CreatePromoCodeRequestObject) (generated.CreatePromoCodeResponseObject, error) {
	return nil, errNotImplemented
}
