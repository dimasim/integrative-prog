package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/dimasim/integrative-prog/internal/domain"
)

// UserRepository mendefinisikan kontrak untuk operasi DB user.
// Interface memudahkan mocking saat unit testing.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindAll(ctx context.Context, limit, offset int) ([]*domain.User, error)
	Update(ctx context.Context, id int64, req *domain.UpdateUserRequest) (*domain.User, error)
	Delete(ctx context.Context, id int64) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// Create — menggunakan $1, $2... parameterized query.
// Tidak ada string concatenation dari input user = SQL Injection TIDAK mungkin.
func (r *userRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (name, email, password, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, name, email, role, created_at, updated_at
	`
	result := &domain.User{}
	err := r.db.QueryRowContext(ctx, query,
		user.Name, user.Email, user.Password, user.Role,
	).Scan(
		&result.ID, &result.Name, &result.Email,
		&result.Role, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("repository.Create: %w", domain.ErrInternalServer)
	}
	return result, nil
}

func (r *userRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `
		SELECT id, name, email, role, created_at, updated_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Name, &user.Email,
		&user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindByID: %w", domain.ErrInternalServer)
	}
	return user, nil
}

// FindByEmail — mengembalikan password hash, digunakan hanya untuk login.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password, role, created_at, updated_at
		FROM users WHERE email = $1 AND deleted_at IS NULL
	`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.Password,
		&user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repository.FindByEmail: %w", domain.ErrInternalServer)
	}
	return user, nil
}

func (r *userRepository) FindAll(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	query := `
		SELECT id, name, email, role, created_at, updated_at
		FROM users WHERE deleted_at IS NULL
		ORDER BY id ASC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("repository.FindAll: %w", domain.ErrInternalServer)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository.FindAll scan: %w", domain.ErrInternalServer)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Update — dynamic PATCH, hanya update field yang tidak kosong.
func (r *userRepository) Update(ctx context.Context, id int64, req *domain.UpdateUserRequest) (*domain.User, error) {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, req.Name)
		argIdx++
	}
	if req.Email != "" {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, req.Email)
		argIdx++
	}
	if req.Role != "" {
		setClauses = append(setClauses, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, req.Role)
		argIdx++
	}
	if len(setClauses) == 0 {
		return r.FindByID(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE users SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, name, email, role, created_at, updated_at
	`, strings.Join(setClauses, ", "), argIdx)

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID, &user.Name, &user.Email,
		&user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("repository.Update: %w", domain.ErrInternalServer)
	}
	return user, nil
}

// Delete — soft delete, data tetap ada di DB dengan deleted_at terisi.
func (r *userRepository) Delete(ctx context.Context, id int64) error {
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repository.Delete: %w", domain.ErrInternalServer)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
