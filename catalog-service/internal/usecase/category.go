package usecase

import (
	"catalog-service/internal/domain"
	"catalog-service/internal/repository"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type CreateCategoryInput struct {
	ParentID *uuid.UUID
	Name     string
}

type UpdateCategoryInput struct {
	ID       uuid.UUID
	ParentID *uuid.UUID
	Name     string
}

type CategoryUseCase interface {
	Create(ctx context.Context, input CreateCategoryInput) (uuid.UUID, error)
	Update(ctx context.Context, input UpdateCategoryInput) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]domain.Category, error)
}

type categoryUseCase struct {
	repo repository.CategoryRepository
}

func NewCategoryUseCase(repo repository.CategoryRepository) CategoryUseCase {
	return &categoryUseCase{repo: repo}
}

func (u *categoryUseCase) Create(ctx context.Context, input CreateCategoryInput) (uuid.UUID, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return uuid.Nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	newCategory := &domain.Category{
		ID:       uuid.New(),
		ParentID: input.ParentID,
		Name:     input.Name,
	}

	if err := u.repo.Create(ctx, newCategory); err != nil {
		return uuid.Nil, fmt.Errorf("usecase.category.Create: %w", err)
	}

	return newCategory.ID, nil
}

func (u *categoryUseCase) Update(ctx context.Context, input UpdateCategoryInput) error {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	if input.ParentID != nil && *input.ParentID == input.ID {
		return fmt.Errorf("%w: category cannot be its own parent", ErrInvalidInput)
	}

	categoryToUpdate := &domain.Category{
		ID:       input.ID,
		ParentID: input.ParentID,
		Name:     input.Name,
	}

	if err := u.repo.Update(ctx, categoryToUpdate); err != nil {
		return fmt.Errorf("usecase.category.Update: %w", err)
	}

	return nil
}

func (u *categoryUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := u.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("usercase.category.Delete: %w", err)
	}
	return nil
}

func (u *categoryUseCase) List(ctx context.Context) ([]domain.Category, error) {
	categories, err := u.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("usecase.category.List: %w", err)
	}
	return categories, nil
}
