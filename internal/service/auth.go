package service

import (
	"context"
	"log/slog"

	"mini_one_api/internal/repository"
)

// ============================================================================
// ИНТЕРФЕЙС AUTH SERVICE
// ============================================================================

type AuthService interface {
	// Authenticate — проверяет токен и возвращает user_id
	Authenticate(ctx context.Context, token string) (string, error)
}

// ============================================================================
// РЕАЛИЗАЦИЯ
// ============================================================================

type authService struct {
	tokenRepo repository.TokenRepository
	logger    *slog.Logger
}

// NewAuthService — конструктор
func NewAuthService(
	tokenRepo repository.TokenRepository,
	logger *slog.Logger,
) AuthService {
	return &authService{
		tokenRepo: tokenRepo,
		logger:    logger,
	}
}

// Authenticate — проверяет токен и возвращает user_id
func (s *authService) Authenticate(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrInvalidToken
	}

	userID, err := s.tokenRepo.GetUserIDByToken(ctx, token)
	if err != nil {
		s.logger.Warn("Authentication failed", "error", err)
		return "", ErrInvalidToken
	}

	s.logger.Debug("Authentication successful", "user_id", userID)
	return userID, nil
}
