package domain

import (
	"math"
	"time"

	"github.com/google/uuid"
)

const MaxNumericCents int64 = 999_999_999_999

type Role string

const (
	RoleUser   Role = "USER"
	RoleSeller Role = "SELLER"
	RoleAdmin  Role = "ADMIN"
)

func (r Role) Valid() bool {
	return r == RoleUser || r == RoleSeller || r == RoleAdmin
}

type Principal struct {
	UserID uuid.UUID
	Role   Role
}

type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "ACTIVE"
	ProductStatusInactive ProductStatus = "INACTIVE"
	ProductStatusArchived ProductStatus = "ARCHIVED"
)

type Product struct {
	ID          uuid.UUID
	SellerID    uuid.UUID
	Name        string
	Description *string
	PriceCents  int64
	Stock       int32
	Category    string
	Status      ProductStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProductInput struct {
	SellerID    uuid.UUID
	Name        string
	Description *string
	PriceCents  int64
	Stock       int32
	Category    string
	Status      ProductStatus
}

type ProductFilter struct {
	Page     int32
	Size     int32
	Status   *ProductStatus
	Category *string
}

type DiscountType string

const (
	DiscountTypePercentage  DiscountType = "PERCENTAGE"
	DiscountTypeFixedAmount DiscountType = "FIXED_AMOUNT"
)

type PromoCode struct {
	ID                      uuid.UUID
	Code                    string
	DiscountType            DiscountType
	DiscountValueHundredths int64
	MinOrderAmountCents     int64
	MaxUses                 int32
	CurrentUses             int32
	ValidFrom               time.Time
	ValidUntil              time.Time
	Active                  bool
}

type PromoCodeInput struct {
	Code                    string
	DiscountType            DiscountType
	DiscountValueHundredths int64
	MinOrderAmountCents     int64
	MaxUses                 int32
	ValidFrom               time.Time
	ValidUntil              time.Time
	Active                  bool
}

type OrderStatus string

const (
	OrderStatusCreated        OrderStatus = "CREATED"
	OrderStatusPaymentPending OrderStatus = "PAYMENT_PENDING"
	OrderStatusPaid           OrderStatus = "PAID"
	OrderStatusShipped        OrderStatus = "SHIPPED"
	OrderStatusCompleted      OrderStatus = "COMPLETED"
	OrderStatusCanceled       OrderStatus = "CANCELED"
)

type Order struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	Status              OrderStatus
	PromoCodeID         *uuid.UUID
	PromoCode           *string
	TotalAmountCents    int64
	DiscountAmountCents int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Items               []OrderItem
}

type OrderItem struct {
	ID                uuid.UUID
	OrderID           uuid.UUID
	ProductID         uuid.UUID
	Quantity          int32
	PriceAtOrderCents int64
}

type OrderItemInput struct {
	ProductID uuid.UUID
	Quantity  int32
}

type UserOperationType string

const (
	UserOperationCreateOrder UserOperationType = "CREATE_ORDER"
	UserOperationUpdateOrder UserOperationType = "UPDATE_ORDER"
)

func Hundredths(value float64) (int64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0, false
	}

	scaled := math.Round(value * 100)
	if scaled > float64(MaxNumericCents) || math.Abs(value*100-scaled) > 1e-7 {
		return 0, false
	}

	return int64(scaled), true
}

func Decimal(hundredths int64) float64 {
	return float64(hundredths) / 100
}
