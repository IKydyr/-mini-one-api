package service

import (
	"context"
	"log/slog"
	"time"

	"mini_one_api/internal/provider"
	"mini_one_api/internal/repository"
)

// ============================================================================
// КОНСТАНТЫ (цены для разных моделей DeepSeek)
// ============================================================================

const (
	// DeepSeek Chat модель
	ModelDeepSeekChat  = "deepseek-chat"
	ModelDeepSeekCoder = "deepseek-coder"

	// Цены за 1M токенов (USD)
	PriceDeepSeekChatInput   = 0.14 // $0.14 за 1M input tokens
	PriceDeepSeekChatOutput  = 0.28 // $0.28 за 1M output tokens
	PriceDeepSeekCoderInput  = 0.14
	PriceDeepSeekCoderOutput = 0.28
)

// ============================================================================
// ИНТЕРФЕЙС CHAT SERVICE (для Handler)
// ============================================================================

type ChatService interface {
	// ProcessChat — обработать запрос к AI (с проверкой баланса)
	ProcessChat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ProcessChatStream — потоковая обработка (для SSE)
	ProcessChatStream(ctx context.Context, req ChatRequest) (<-chan string, <-chan error)
}

// ============================================================================
// РЕАЛИЗАЦИЯ (приватная структура)
// ============================================================================

type chatService struct {
	userRepo   repository.UserRepository
	tokenRepo  repository.TokenRepository
	chargeRepo repository.ChargeRepository
	deepseek   *provider.DeepSeekProvider
	logger     *slog.Logger
}

// NewChatService — конструктор
func NewChatService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	chargeRepo repository.ChargeRepository,
	deepseek *provider.DeepSeekProvider,
	logger *slog.Logger,
) ChatService {
	return &chatService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		chargeRepo: chargeRepo,
		deepseek:   deepseek,
		logger:     logger,
	}
}

// ProcessChat — обычный (не потоковый) запрос
func (s *chatService) ProcessChat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	s.logger.Debug("Processing chat", "user_id", req.UserID, "model", req.Model)

	// =========================================================================
	// ШАГ 1: ВАЛИДАЦИЯ ВХОДНЫХ ДАННЫХ
	// =========================================================================
	if req.UserID == "" {
		return nil, ErrInvalidToken
	}
	if req.Model == "" {
		req.Model = ModelDeepSeekChat
	}
	if len(req.Messages) == 0 {
		return nil, NewBusinessError("NO_MESSAGES", "No messages provided", 400, nil)
	}

	// =========================================================================
	// ШАГ 2: ПРОВЕРКА БАЛАНСА ПОЛЬЗОВАТЕЛЯ
	// =========================================================================
	balance, err := s.userRepo.GetBalance(ctx, req.UserID)
	if err != nil {
		s.logger.Error("Failed to get balance", "error", err)
		return nil, NewBusinessError("BALANCE_ERROR", "Failed to check balance", 500, err)
	}

	// БИЗНЕС-ПРАВИЛО: минимальный баланс для запроса
	minRequiredBalance := 0.01
	if balance < minRequiredBalance {
		s.logger.Warn("Insufficient balance",
			"user_id", req.UserID,
			"balance", balance,
			"required", minRequiredBalance,
		)
		return nil, ErrInsufficientBalance
	}

	// =========================================================================
	// ШАГ 3: РАСЧЁТ СТОИМОСТИ ЗАПРОСА
	// =========================================================================
	estimatedTokens := s.estimateTokens(req.Messages)
	estimatedCost := s.calculateCost(req.Model, estimatedTokens)

	s.logger.Debug("Cost calculated",
		"estimated_tokens", estimatedTokens,
		"estimated_cost", estimatedCost,
	)

	// =========================================================================
	// ШАГ 4: ОТПРАВКА ЗАПРОСА К DEEPSEEK
	// =========================================================================
	deepseekReq := provider.DeepSeekRequest{
		Model:    req.Model,
		Messages: s.convertMessages(req.Messages),
		Stream:   false,
	}

	deepseekResp, err := s.deepseek.ChatCompletion(ctx, deepseekReq)
	if err != nil {
		s.logger.Error("DeepSeek API error", "error", err)
		return nil, NewBusinessError("PROVIDER_ERROR", "AI provider error", 502, err)
	}

	// =========================================================================
	// ШАГ 5: РАСЧЁТ ФАКТИЧЕСКОЙ СТОИМОСТИ
	// =========================================================================
	actualTokens := deepseekResp.Usage.TotalTokens
	actualCost := s.calculateCost(req.Model, actualTokens)

	// =========================================================================
	// ШАГ 6: АТОМАРНОЕ СПИСАНИЕ СРЕДСТВ (с проверкой баланса)
	// =========================================================================
	// Используем атомарную операцию: списываем только если хватает
	newBalance, err := s.userRepo.DeductBalance(ctx, req.UserID, actualCost)
	if err != nil {
		s.logger.Error("Failed to deduct balance", "error", err)
		return nil, ErrInsufficientBalance
	}

	s.logger.Debug("Balance deducted",
		"user_id", req.UserID,
		"amount", actualCost,
		"new_balance", newBalance,
	)
	// =========================================================================
	// ШАГ 7: ОБНОВЛЕНИЕ СЧЁТЧИКА ТОКЕНОВ
	// =========================================================================
	if err := s.userRepo.AddTokens(ctx, req.UserID, int64(actualTokens)); err != nil {
		s.logger.Error("Failed to add tokens", "error", err)
		// Не возвращаем ошибку, только логируем (баланс уже списан)
	}

	// =========================================================================
	// ШАГ 8: ЗАПИСЬ ИСТОРИИ
	// =========================================================================
	chargeRecord := repository.ChargeRecord{
		UserID:       req.UserID,
		Model:        req.Model,
		Provider:     "deepseek",
		InputTokens:  deepseekResp.Usage.PromptTokens,
		OutputTokens: deepseekResp.Usage.CompletionTokens,
		CostUSD:      actualCost,
	}
	if err := s.chargeRepo.RecordCharge(ctx, chargeRecord); err != nil {
		s.logger.Error("Failed to record charge", "error", err)
		// Не возвращаем ошибку, только логируем
	}

	// =========================================================================
	// ШАГ 9: ФОРМИРОВАНИЕ ОТВЕТА
	// =========================================================================
	response := &ChatResponse{
		Content:      deepseekResp.Choices[0].Message.Content,
		TokensUsed:   actualTokens,
		CostUSD:      actualCost,
		ResponseTime: time.Since(startTime),
	}

	s.logger.Info("Chat processed",
		"user_id", req.UserID,
		"tokens", actualTokens,
		"cost", actualCost,
		"duration_ms", response.ResponseTime.Milliseconds(),
	)

	return response, nil
}

// ProcessChatStream — потоковый ответ (для SSE)
func (s *chatService) ProcessChatStream(ctx context.Context, req ChatRequest) (<-chan string, <-chan error) {
	messages := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(messages)
		defer close(errChan)

		// Проверка баланса (упрощённо, без списания до конца потока)
		balance, err := s.userRepo.GetBalance(ctx, req.UserID)
		if err != nil {
			errChan <- ErrInternal
			return
		}

		if balance < 0.01 {
			errChan <- ErrInsufficientBalance
			return
		}

		// Отправка запроса к DeepSeek в потоковом режиме
		deepseekReq := provider.DeepSeekRequest{
			Model:    req.Model,
			Messages: s.convertMessages(req.Messages),
			Stream:   true,
		}

		stream, err := s.deepseek.ChatCompletionStream(ctx, deepseekReq)
		if err != nil {
			errChan <- NewBusinessError("PROVIDER_ERROR", "Failed to start stream", 502, err)
			return
		}
		defer stream.Close()

		// Передаём чанки в канал
		for chunk := range stream {
			select {
			case messages <- chunk:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}

		// TODO: после окончания потока — списать средства
		// Для упрощения пока пропускаем
	}()

	return messages, errChan
}

// ============================================================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ (приватные)
// ============================================================================

// estimateTokens — грубая оценка количества токенов
func (s *chatService) estimateTokens(messages []ChatMessage) int {
	total := 0
	for _, msg := range messages {
		// 1 токен ≈ 4 символа (грубая оценка)
		total += len(msg.Content) / 4
	}
	// Системный оверхед
	total += len(messages) * 4
	return total
}

// calculateCost — расчёт стоимости запроса
func (s *chatService) calculateCost(model string, tokens int) float64 {
	// Цена за 1K токенов (USD)
	var pricePer1K float64

	switch model {
	case ModelDeepSeekChat:
		pricePer1K = PriceDeepSeekChatInput / 1000
	case ModelDeepSeekCoder:
		pricePer1K = PriceDeepSeekCoderInput / 1000
	default:
		pricePer1K = 0.00014 // 0.14 / 1000
	}

	return float64(tokens) * pricePer1K
}

// convertMessages — конвертация Business DTO → Provider DTO
func (s *chatService) convertMessages(messages []ChatMessage) []provider.DeepSeekMessage {
	result := make([]provider.DeepSeekMessage, len(messages))
	for i, msg := range messages {
		result[i] = provider.DeepSeekMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}
