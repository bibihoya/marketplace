package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace/commerce-service/internal/domain"
)

type Repository interface {
	Ping(context.Context) error
	WithTx(context.Context, func(Transaction) error) error

	ListProducts(context.Context, domain.ProductFilter) ([]domain.Product, int64, error)
	GetProduct(context.Context, uuid.UUID) (domain.Product, error)
	CreateProduct(context.Context, domain.ProductInput) (domain.Product, error)
	UpdateProduct(context.Context, uuid.UUID, domain.ProductInput) (domain.Product, error)
	ArchiveProduct(context.Context, uuid.UUID) (domain.Product, error)
	CreatePromoCode(context.Context, domain.PromoCodeInput) (domain.PromoCode, error)
	GetOrder(context.Context, uuid.UUID) (domain.Order, error)
	ListOrderItems(context.Context, uuid.UUID) ([]domain.OrderItem, error)
}

type Transaction interface {
	LockUser(context.Context, uuid.UUID) error
	LastUserOperation(context.Context, uuid.UUID, domain.UserOperationType) (*time.Time, error)
	HasActiveOrder(context.Context, uuid.UUID) (bool, error)
	LockProducts(context.Context, []uuid.UUID) (map[uuid.UUID]domain.Product, error)
	SetProductStock(context.Context, uuid.UUID, int32) error

	LockPromoCodeByCode(context.Context, string) (domain.PromoCode, error)
	LockPromoCodeByID(context.Context, uuid.UUID) (domain.PromoCode, error)
	ChangePromoCodeUses(context.Context, uuid.UUID, int32) error

	CreateOrder(context.Context, domain.Order) (domain.Order, error)
	FindOrderOwner(context.Context, uuid.UUID) (uuid.UUID, error)
	LockOrder(context.Context, uuid.UUID) (domain.Order, error)
	UpdateOrder(context.Context, domain.Order) (domain.Order, error)
	SetOrderStatus(context.Context, uuid.UUID, domain.OrderStatus) (domain.Order, error)
	ListOrderItems(context.Context, uuid.UUID) ([]domain.OrderItem, error)
	DeleteOrderItems(context.Context, uuid.UUID) error
	CreateOrderItems(context.Context, uuid.UUID, []domain.OrderItem) ([]domain.OrderItem, error)
	RecordUserOperation(context.Context, uuid.UUID, domain.UserOperationType) error
}
