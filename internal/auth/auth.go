package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/user/gater/internal/model"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

var (
	ErrInvalidAPIKey  = errors.New("invalid API key")
	ErrInvalidCreds   = errors.New("invalid email or password")
	ErrEmailTaken     = errors.New("email already registered")
)

func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	APIKey string `json:"api_key,omitempty"`
}

func toUserResponse(u *model.User) UserResponse {
	return UserResponse{
		ID:     u.ID,
		Email:  u.Email,
		Name:   u.Name,
		APIKey: u.APIKey,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*AuthResponse, error) {
	var existing model.User
	if err := s.db.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		return nil, ErrEmailTaken
	}

	hash, err := HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := model.User{
		Email:        input.Email,
		PasswordHash: hash,
		Name:         input.Name,
		APIKey:       generateAPIKey(),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: user.APIKey,
		User:  toUserResponse(&user),
	}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	var user model.User
	if err := s.db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		return nil, ErrInvalidCreds
	}

	if !checkPassword(user.PasswordHash, input.Password) {
		return nil, ErrInvalidCreds
	}

	return &AuthResponse{
		Token: user.APIKey,
		User:  toUserResponse(&user),
	}, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (*UserResponse, error) {
	var user model.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	resp := toUserResponse(&user)
	return &resp, nil
}
