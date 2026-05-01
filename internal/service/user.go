package service

import (
	"context"
	"errors"
	"log/slog"

	"mini_one_api/internal/repository"
)

// ============================================================================
// ИНТЕРФЕЙС USER SERVICE (для Handler)
// ============================================================================

type UserService interface {
	// GetUserInfo — получить информацию о пользователе
	GetUserInfo(ctx context.Context, req GetUserInfoRequest) (*UserInfoResponse, error)
}

// ============================================================================
// РЕАЛИЗАЦИЯ (приватная структура)
// ============================================================================

type userService struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
	logger    *slog.Logger
}

// NewUserService — конструктор (возвращает интерфейс!)
func NewUserService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	logger *slog.Logger,
) UserService {
	return &userService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		logger:    logger,
	}
}

// GetUserInfo — реализация бизнес-логики получения информации о пользователе
func (s *userService) GetUserInfo(ctx context.Context, req GetUserInfoRequest) (*UserInfoResponse, error) {
	s.logger.Debug("Getting user info", "user_id", req.UserID)

	// =========================================================================
	// ШАГ 1: ВАЛИДАЦИЯ ВХОДНЫХ ДАННЫХ (бизнес-правило)
	// =========================================================================
	if req.UserID == "" {
		return nil, ErrInvalidToken
	}

	// =========================================================================
	// ШАГ 2: ПОЛУЧЕНИЕ ДАННЫХ ЧЕРЕЗ РЕПОЗИТОРИЙ
	// =========================================================================

	// Получаем пользователя из БД
	userDB, err := s.userRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		s.logger.Error("Failed to get user", "user_id", req.UserID, "error", err)

		// Превращаем ошибку репозитория в бизнес-ошибку
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, NewBusinessError("DB_ERROR", "Database error", 500, err)
	}

	// =========================================================================
	// ШАГ 3: ФОРМИРОВАНИЕ ОТВЕТА (маппинг Database DTO → Business DTO)
	// =========================================================================
	response := &UserInfoResponse{
		UserID:      userDB.UserID,
		BalanceUSD:  userDB.BalanceUSD,
		TotalTokens: userDB.TotalTokensUsed,
	}

	s.logger.Info("User info retrieved",
		"user_id", req.UserID,
		"balance", response.BalanceUSD,
		"tokens", response.TotalTokens,
	)

	return response, nil
}
