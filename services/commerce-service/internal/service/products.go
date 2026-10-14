package service

import (
	"context"

	"github.com/google/uuid"

	"marketplace/commerce-service/internal/domain"
	"marketplace/commerce-service/internal/repository"
)

func (s *Service) ListProducts(ctx context.Context, principal domain.Principal, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	if err := requireRole(principal, domain.RoleUser, domain.RoleSeller, domain.RoleAdmin); err != nil {
		return nil, 0, err
	}

	products, total, err := s.repository.ListProducts(ctx, filter)
	if err != nil {
		return nil, 0, internalError(err)
	}
	return products, total, nil
}

func (s *Service) GetProduct(ctx context.Context, principal domain.Principal, id uuid.UUID) (domain.Product, error) {
	if err := requireRole(principal, domain.RoleUser, domain.RoleSeller, domain.RoleAdmin); err != nil {
		return domain.Product{}, err
	}

	product, err := s.repository.GetProduct(ctx, id)
	if repository.IsNotFound(err) {
		return domain.Product{}, productNotFound()
	}
	if err != nil {
		return domain.Product{}, internalError(err)
	}
	return product, nil
}

func (s *Service) CreateProduct(ctx context.Context, principal domain.Principal, input domain.ProductInput) (domain.Product, error) {
	if err := requireRole(principal, domain.RoleSeller, domain.RoleAdmin); err != nil {
		return domain.Product{}, err
	}

	if principal.Role == domain.RoleSeller {
		if input.SellerID != uuid.Nil && input.SellerID != principal.UserID {
			return domain.Product{}, newError(CodeAccessDenied, "Seller can create only own products", nil)
		}
		input.SellerID = principal.UserID
	} else if input.SellerID == uuid.Nil {
		input.SellerID = principal.UserID
	}

	product, err := s.repository.CreateProduct(ctx, input)
	if err != nil {
		return domain.Product{}, internalError(err)
	}
	return product, nil
}

func (s *Service) UpdateProduct(ctx context.Context, principal domain.Principal, id uuid.UUID, input domain.ProductInput) (domain.Product, error) {
	if err := requireRole(principal, domain.RoleSeller, domain.RoleAdmin); err != nil {
		return domain.Product{}, err
	}

	existing, err := s.repository.GetProduct(ctx, id)
	if repository.IsNotFound(err) {
		return domain.Product{}, productNotFound()
	}
	if err != nil {
		return domain.Product{}, internalError(err)
	}
	if principal.Role == domain.RoleSeller && existing.SellerID != principal.UserID {
		return domain.Product{}, newError(CodeAccessDenied, "Seller can update only own products", nil)
	}

	input.SellerID = existing.SellerID
	product, err := s.repository.UpdateProduct(ctx, id, input)
	if repository.IsNotFound(err) {
		return domain.Product{}, productNotFound()
	}
	if err != nil {
		return domain.Product{}, internalError(err)
	}
	return product, nil
}

func (s *Service) ArchiveProduct(ctx context.Context, principal domain.Principal, id uuid.UUID) (domain.Product, error) {
	if err := requireRole(principal, domain.RoleSeller, domain.RoleAdmin); err != nil {
		return domain.Product{}, err
	}

	existing, err := s.repository.GetProduct(ctx, id)
	if repository.IsNotFound(err) {
		return domain.Product{}, productNotFound()
	}
	if err != nil {
		return domain.Product{}, internalError(err)
	}
	if principal.Role == domain.RoleSeller && existing.SellerID != principal.UserID {
		return domain.Product{}, newError(CodeAccessDenied, "Seller can archive only own products", nil)
	}

	product, err := s.repository.ArchiveProduct(ctx, id)
	if repository.IsNotFound(err) {
		return domain.Product{}, productNotFound()
	}
	if err != nil {
		return domain.Product{}, internalError(err)
	}
	return product, nil
}
