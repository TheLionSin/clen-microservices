package postgres

import (
	"catalog-service/internal/domain"
	"catalog-service/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type categoryRepo struct {
	pool *pgxpool.Pool
}

func NewCategoryRepo(pool *pgxpool.Pool) repository.CategoryRepository {
	return &categoryRepo{pool: pool}
}

func (r *categoryRepo) Create(ctx context.Context, c *domain.Category) error {
	query := `INSERT INTO categories (id, parent_id, name) VALUES ($1, $2, $3)`

	// pgx автоматически конвертирует указатель *uuid.UUID в базу:
	// если nil, то запишет NULL. Если есть значение, запишет UUID.
	_, err := r.pool.Exec(ctx, query, c.ID, c.ParentID, c.Name)
	if err != nil {
		return fmt.Errorf("postgres.categoryRepo.Create: %w", err)
	}
	return nil
}

func (r *categoryRepo) Update(ctx context.Context, c *domain.Category) error {
	query := `UPDATE categories SET parent_id = $1, name = $2
				WHERE id = $3`

	cmdTag, err := r.pool.Exec(ctx, query, c.ParentID, c.Name, c.ID)
	if err != nil {
		return fmt.Errorf("postgres.categoryRepo.Update: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}

func (r *categoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM categories WHERE id = $1`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrCategoryInUse
		}
		return fmt.Errorf("postgres.categoryRepo.Delete: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}

	return nil
}

func (r *categoryRepo) List(ctx context.Context) ([]domain.Category, error) {
	query := `SELECT id, parent_id, name FROM categories ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres.categoryRepo.List: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.ParentID, &c.Name); err != nil {
			return nil, fmt.Errorf("postgres.categoryRepo.List scan: %w", err)
		}
		categories = append(categories, c)
	}

	return categories, nil
}
