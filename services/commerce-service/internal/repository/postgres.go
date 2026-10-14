package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"marketplace/commerce-service/internal/domain"
)

type Postgres struct {
	pool *pgxpool.Pool
}

type postgresTx struct {
	tx pgx.Tx
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (r *Postgres) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Postgres) WithTx(ctx context.Context, fn func(Transaction) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(&postgresTx{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *Postgres) ListProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	status := nullableProductStatus(filter.Status)
	category := nullableString(filter.Category)

	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM products
		WHERE ($1::product_status IS NULL OR status = $1)
		  AND ($2::text IS NULL OR category = $2)
	`, status, category).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	offset := int64(filter.Page) * int64(filter.Size)
	rows, err := r.pool.Query(ctx, `
		SELECT id, seller_id, name, description, (price * 100)::bigint,
		       stock, category, status::text, created_at, updated_at
		FROM products
		WHERE ($1::product_status IS NULL OR status = $1)
		  AND ($2::text IS NULL OR category = $2)
		ORDER BY created_at DESC, id
		LIMIT $3 OFFSET $4
	`, status, category, filter.Size, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0, filter.Size)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate products: %w", err)
	}

	return products, total, nil
}

func (r *Postgres) GetProduct(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return scanProduct(r.pool.QueryRow(ctx, `
		SELECT id, seller_id, name, description, (price * 100)::bigint,
		       stock, category, status::text, created_at, updated_at
		FROM products
		WHERE id = $1
	`, id))
}

func (r *Postgres) CreateProduct(ctx context.Context, input domain.ProductInput) (domain.Product, error) {
	return scanProduct(r.pool.QueryRow(ctx, `
		INSERT INTO products (seller_id, name, description, price, stock, category, status)
		VALUES ($1, $2, $3, $4::bigint / 100.0, $5, $6, $7)
		RETURNING id, seller_id, name, description, (price * 100)::bigint,
		          stock, category, status::text, created_at, updated_at
	`, input.SellerID, input.Name, input.Description, input.PriceCents, input.Stock, input.Category, input.Status))
}

func (r *Postgres) UpdateProduct(ctx context.Context, id uuid.UUID, input domain.ProductInput) (domain.Product, error) {
	return scanProduct(r.pool.QueryRow(ctx, `
		UPDATE products
		SET name = $2,
		    description = $3,
		    price = $4::bigint / 100.0,
		    stock = $5,
		    category = $6,
		    status = $7
		WHERE id = $1
		RETURNING id, seller_id, name, description, (price * 100)::bigint,
		          stock, category, status::text, created_at, updated_at
	`, id, input.Name, input.Description, input.PriceCents, input.Stock, input.Category, input.Status))
}

func (r *Postgres) ArchiveProduct(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return scanProduct(r.pool.QueryRow(ctx, `
		UPDATE products
		SET status = 'ARCHIVED'
		WHERE id = $1
		RETURNING id, seller_id, name, description, (price * 100)::bigint,
		          stock, category, status::text, created_at, updated_at
	`, id))
}

func (r *Postgres) CreatePromoCode(ctx context.Context, input domain.PromoCodeInput) (domain.PromoCode, error) {
	return scanPromoCode(r.pool.QueryRow(ctx, `
		INSERT INTO promo_codes (
			code, discount_type, discount_value, min_order_amount,
			max_uses, valid_from, valid_until, active
		)
		VALUES ($1, $2, $3::bigint / 100.0, $4::bigint / 100.0, $5, $6, $7, $8)
		RETURNING id, code, discount_type::text, (discount_value * 100)::bigint,
		          (min_order_amount * 100)::bigint, max_uses, current_uses,
		          valid_from, valid_until, active
	`, input.Code, input.DiscountType, input.DiscountValueHundredths,
		input.MinOrderAmountCents, input.MaxUses, input.ValidFrom, input.ValidUntil, input.Active))
}

func (r *Postgres) GetOrder(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	return scanOrder(r.pool.QueryRow(ctx, orderSelect+` WHERE o.id = $1`, id))
}

func (r *Postgres) ListOrderItems(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	return listOrderItems(ctx, r.pool, orderID)
}

func (r *postgresTx) LockUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, userID)
	if err != nil {
		return fmt.Errorf("lock user operations: %w", err)
	}
	return nil
}

func (r *postgresTx) LastUserOperation(ctx context.Context, userID uuid.UUID, operation domain.UserOperationType) (*time.Time, error) {
	var createdAt time.Time
	err := r.tx.QueryRow(ctx, `
		SELECT created_at
		FROM user_operations
		WHERE user_id = $1 AND operation_type = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, operation).Scan(&createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get last user operation: %w", err)
	}
	return &createdAt, nil
}

func (r *postgresTx) HasActiveOrder(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM orders
			WHERE user_id = $1 AND status IN ('CREATED', 'PAYMENT_PENDING')
		)
	`, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check active order: %w", err)
	}
	return exists, nil
}

func (r *postgresTx) LockProducts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.Product, error) {
	products := make(map[uuid.UUID]domain.Product, len(ids))
	for _, id := range ids {
		product, err := scanProduct(r.tx.QueryRow(ctx, `
			SELECT id, seller_id, name, description, (price * 100)::bigint,
			       stock, category, status::text, created_at, updated_at
			FROM products
			WHERE id = $1
			FOR UPDATE
		`, id))
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		products[id] = product
	}
	return products, nil
}

func (r *postgresTx) SetProductStock(ctx context.Context, productID uuid.UUID, stock int32) error {
	_, err := r.tx.Exec(ctx, `UPDATE products SET stock = $2 WHERE id = $1`, productID, stock)
	if err != nil {
		return fmt.Errorf("update product stock: %w", err)
	}
	return nil
}

func (r *postgresTx) LockPromoCodeByCode(ctx context.Context, code string) (domain.PromoCode, error) {
	return scanPromoCode(r.tx.QueryRow(ctx, `
		SELECT id, code, discount_type::text, (discount_value * 100)::bigint,
		       (min_order_amount * 100)::bigint, max_uses, current_uses,
		       valid_from, valid_until, active
		FROM promo_codes
		WHERE code = $1
		FOR UPDATE
	`, code))
}

func (r *postgresTx) LockPromoCodeByID(ctx context.Context, id uuid.UUID) (domain.PromoCode, error) {
	return scanPromoCode(r.tx.QueryRow(ctx, `
		SELECT id, code, discount_type::text, (discount_value * 100)::bigint,
		       (min_order_amount * 100)::bigint, max_uses, current_uses,
		       valid_from, valid_until, active
		FROM promo_codes
		WHERE id = $1
		FOR UPDATE
	`, id))
}

func (r *postgresTx) ChangePromoCodeUses(ctx context.Context, id uuid.UUID, delta int32) error {
	_, err := r.tx.Exec(ctx, `
		UPDATE promo_codes
		SET current_uses = current_uses + $2
		WHERE id = $1
	`, id, delta)
	if err != nil {
		return fmt.Errorf("change promo code uses: %w", err)
	}
	return nil
}

func (r *postgresTx) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	return scanOrder(r.tx.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO orders (user_id, status, promo_code_id, total_amount, discount_amount)
			VALUES ($1, $2, $3, $4::bigint / 100.0, $5::bigint / 100.0)
			RETURNING *
		)
		SELECT i.id, i.user_id, i.status::text, i.promo_code_id, p.code,
		       (i.total_amount * 100)::bigint, (i.discount_amount * 100)::bigint,
		       i.created_at, i.updated_at
		FROM inserted i
		LEFT JOIN promo_codes p ON p.id = i.promo_code_id
	`, order.UserID, order.Status, order.PromoCodeID, order.TotalAmountCents, order.DiscountAmountCents))
}

func (r *postgresTx) FindOrderOwner(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.tx.QueryRow(ctx, `SELECT user_id FROM orders WHERE id = $1`, id).Scan(&userID)
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (r *postgresTx) LockOrder(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	return scanOrder(r.tx.QueryRow(ctx, orderSelect+` WHERE o.id = $1 FOR UPDATE OF o`, id))
}

func (r *postgresTx) UpdateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	return scanOrder(r.tx.QueryRow(ctx, `
		WITH updated AS (
			UPDATE orders
			SET promo_code_id = $2,
			    total_amount = $3::bigint / 100.0,
			    discount_amount = $4::bigint / 100.0
			WHERE id = $1
			RETURNING *
		)
		SELECT u.id, u.user_id, u.status::text, u.promo_code_id, p.code,
		       (u.total_amount * 100)::bigint, (u.discount_amount * 100)::bigint,
		       u.created_at, u.updated_at
		FROM updated u
		LEFT JOIN promo_codes p ON p.id = u.promo_code_id
	`, order.ID, order.PromoCodeID, order.TotalAmountCents, order.DiscountAmountCents))
}

func (r *postgresTx) SetOrderStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (domain.Order, error) {
	return scanOrder(r.tx.QueryRow(ctx, `
		WITH updated AS (
			UPDATE orders
			SET status = $2
			WHERE id = $1
			RETURNING *
		)
		SELECT u.id, u.user_id, u.status::text, u.promo_code_id, p.code,
		       (u.total_amount * 100)::bigint, (u.discount_amount * 100)::bigint,
		       u.created_at, u.updated_at
		FROM updated u
		LEFT JOIN promo_codes p ON p.id = u.promo_code_id
	`, id, status))
}

func (r *postgresTx) ListOrderItems(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	return listOrderItems(ctx, r.tx, orderID)
}

func (r *postgresTx) DeleteOrderItems(ctx context.Context, orderID uuid.UUID) error {
	_, err := r.tx.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, orderID)
	if err != nil {
		return fmt.Errorf("delete order items: %w", err)
	}
	return nil
}

func (r *postgresTx) CreateOrderItems(ctx context.Context, orderID uuid.UUID, items []domain.OrderItem) ([]domain.OrderItem, error) {
	created := make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
		var result domain.OrderItem
		err := r.tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, product_id, quantity, price_at_order)
			VALUES ($1, $2, $3, $4::bigint / 100.0)
			RETURNING id, order_id, product_id, quantity, (price_at_order * 100)::bigint
		`, orderID, item.ProductID, item.Quantity, item.PriceAtOrderCents).Scan(
			&result.ID,
			&result.OrderID,
			&result.ProductID,
			&result.Quantity,
			&result.PriceAtOrderCents,
		)
		if err != nil {
			return nil, fmt.Errorf("create order item: %w", err)
		}
		created = append(created, result)
	}
	return created, nil
}

func (r *postgresTx) RecordUserOperation(ctx context.Context, userID uuid.UUID, operation domain.UserOperationType) error {
	_, err := r.tx.Exec(ctx, `
		INSERT INTO user_operations (user_id, operation_type)
		VALUES ($1, $2)
	`, userID, operation)
	if err != nil {
		return fmt.Errorf("record user operation: %w", err)
	}
	return nil
}

const orderSelect = `
	SELECT o.id, o.user_id, o.status::text, o.promo_code_id, p.code,
	       (o.total_amount * 100)::bigint, (o.discount_amount * 100)::bigint,
	       o.created_at, o.updated_at
	FROM orders o
	LEFT JOIN promo_codes p ON p.id = o.promo_code_id
`

type rowScanner interface {
	Scan(...any) error
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func scanProduct(row rowScanner) (domain.Product, error) {
	var product domain.Product
	var status string
	err := row.Scan(
		&product.ID,
		&product.SellerID,
		&product.Name,
		&product.Description,
		&product.PriceCents,
		&product.Stock,
		&product.Category,
		&status,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return domain.Product{}, err
	}
	product.Status = domain.ProductStatus(status)
	return product, nil
}

func scanPromoCode(row rowScanner) (domain.PromoCode, error) {
	var promo domain.PromoCode
	var discountType string
	err := row.Scan(
		&promo.ID,
		&promo.Code,
		&discountType,
		&promo.DiscountValueHundredths,
		&promo.MinOrderAmountCents,
		&promo.MaxUses,
		&promo.CurrentUses,
		&promo.ValidFrom,
		&promo.ValidUntil,
		&promo.Active,
	)
	if err != nil {
		return domain.PromoCode{}, err
	}
	promo.DiscountType = domain.DiscountType(discountType)
	return promo, nil
}

func scanOrder(row rowScanner) (domain.Order, error) {
	var order domain.Order
	var status string
	err := row.Scan(
		&order.ID,
		&order.UserID,
		&status,
		&order.PromoCodeID,
		&order.PromoCode,
		&order.TotalAmountCents,
		&order.DiscountAmountCents,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return domain.Order{}, err
	}
	order.Status = domain.OrderStatus(status)
	return order, nil
}

func listOrderItems(ctx context.Context, db queryer, orderID uuid.UUID) ([]domain.OrderItem, error) {
	rows, err := db.Query(ctx, `
		SELECT id, order_id, product_id, quantity, (price_at_order * 100)::bigint
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.PriceAtOrderCents); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}
	return items, nil
}

func nullableProductStatus(status *domain.ProductStatus) any {
	if status == nil {
		return nil
	}
	return string(*status)
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
