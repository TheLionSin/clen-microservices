package postgres

import (
	"catalog-service/internal/domain"
	"catalog-service/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type productRepo struct {
	pool *pgxpool.Pool
}

func NewProductRepo(pool *pgxpool.Pool) repository.ProductRepository {
	return &productRepo{
		pool: pool,
	}
}

func (r *productRepo) Create(ctx context.Context, p *domain.Product) error {
	query := `INSERT INTO products (id,category_id,name,description,price,stock,image_urls, created_at, updated_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
				`
	_, err := r.pool.Exec(ctx, query, p.ID, p.CategoryID, p.Name, p.Description, p.Price,
		p.Stock, p.ImageURLs, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres.productRepo.Create: %w", err)
	}

	return nil
}

func (r *productRepo) Update(ctx context.Context, p *domain.Product) error {
	query := `UPDATE products SET category_id = $1, name = $2, description = $3,
                    price = $4, stock = $5, image_urls = $6, updated_at = $7
                    WHERE id = $8`

	cmdTag, err := r.pool.Exec(ctx, query, p.CategoryID, p.Name, p.Description,
		p.Price, p.Stock, p.ImageURLs, p.UpdatedAt, p.ID)
	if err != nil {
		return fmt.Errorf("postgres.productRepo.Update: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *productRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE from products WHERE id = $1`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("postgres.productRepo.Delete: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *productRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	query := `SELECT id,category_id,name,description,price,stock,image_urls,created_at,updated_at FROM products WHERE id = $1`

	var p domain.Product
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURLs, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("postgres.productRepo.GetByID: %w", err)
	}

	return &p, nil
}

func (r *productRepo) List(ctx context.Context, limit, offset int) ([]domain.Product, error) {
	query := `SELECT id,category_id,name,description,price,stock,image_urls,created_at,updated_at FROM products
				ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("postgres.productRepo.List: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0, limit)

	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURLs, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres.productRepo.List scan: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.productRepo.List rows err: %w", err)
	}

	return products, err
}

func (r *productRepo) DecrementStock(ctx context.Context, id uuid.UUID, quantity int) error {
	// Атомарный UPDATE: вычитаем quantity только если текущий stock >= quantity
	query := `
			UPDATE products SET stock = stock - $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND stock >= $1`

	cmdTag, err := r.pool.Exec(ctx, query, quantity, id)
	if err != nil {
		return fmt.Errorf("postgres.productRepo.DecrementStock: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotEnoughStock
	}

	return nil
}
