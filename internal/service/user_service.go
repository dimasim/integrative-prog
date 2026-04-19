package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/dimasim/integrative-prog/internal/config"
	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/repository"
)

// UserService mendefinisikan business logic untuk user.
type UserService interface {
	Register(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetAll(ctx context.Context, limit, offset int) ([]*domain.User, error)
	Update(ctx context.Context, id int64, req *domain.UpdateUserRequest) (*domain.User, error)
	Delete(ctx context.Context, id int64) error
}

type userService struct {
	repo   repository.UserRepository
	jwtCfg config.JWTConfig
}

func NewUserService(repo repository.UserRepository, jwtCfg config.JWTConfig) UserService {
	return &userService{repo: repo, jwtCfg: jwtCfg}
}

// Register — hash password sebelum menyimpan ke DB.
func (s *userService) Register(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("service.Register hash: %w", domain.ErrInternalServer)
	}

	user := &domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashed),
		Role:     req.Role,
	}
	return s.repo.Create(ctx, user)
}

// Login — verifikasi credentials dan generate JWT token.
func (s *userService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Selalu kembalikan ErrInvalidCreds untuk mencegah user enumeration
		return nil, domain.ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCreds
	}

	token, err := s.generateJWT(user)
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	return &domain.LoginResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresIn: s.jwtCfg.ExpiryHours,
	}, nil
}

func (s *userService) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *userService) GetAll(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.FindAll(ctx, limit, offset)
}

func (s *userService) Update(ctx context.Context, id int64, req *domain.UpdateUserRequest) (*domain.User, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *userService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// ─── JWT Helper ───────────────────────────────────────────────────────────────

type JWTClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (s *userService) generateJWT(user *domain.User) (string, error) {
	claims := JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.jwtCfg.ExpiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "go-restapi",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtCfg.Secret))
}
