package service

import (
	"time"
)

// ============================================================================
// DTO ДЛЯ ПОЛЬЗОВАТЕЛЯ (User Service)
// ============================================================================

// GetUserInfoRequest — запрос на получение информации о пользователе
type GetUserInfoRequest struct {
	UserID string // ID пользователя (из токена)
}

// UserInfoResponse — ответ с информацией о пользователе
type UserInfoResponse struct {
	UserID      string  `json:"user_id"`      // ID пользователя
	BalanceUSD  float64 `json:"balance_usd"`  // баланс в долларах
	TotalTokens int64   `json:"total_tokens"` // всего потрачено токенов
}

// ============================================================================
// DTO ДЛЯ ЧАТА (Chat Service)
// ============================================================================

// ChatRequest — запрос на отправку сообщения в AI
type ChatRequest struct {
	UserID   string        // ID пользователя (из токена)
	Model    string        // модель (deepseek-chat, deepseek-coder)
	Messages []ChatMessage // история сообщений
	Stream   bool          // потоковый ответ?
}

// ChatResponse — ответ от AI
type ChatResponse struct {
	Content      string        `json:"content"`       // текст ответа
	TokensUsed   int           `json:"tokens_used"`   // сколько токенов потрачено
	CostUSD      float64       `json:"cost_usd"`      // стоимость в долларах
	ResponseTime time.Duration `json:"response_time"` // время ответа
}

// ChatMessage — одно сообщение в диалоге
type ChatMessage struct {
	Role    string `json:"role"`    // "user" или "assistant"
	Content string `json:"content"` // текст сообщения
}

// ============================================================================
// DTO ДЛЯ АУТЕНТИФИКАЦИИ (Auth Service)
// ============================================================================

// AuthRequest — запрос на аутентификацию
type AuthRequest struct {
	Token string // API ключ (sk-xxxxx)
}

// AuthResponse — результат аутентификации
type AuthResponse struct {
	UserID    string     `json:"user_id"`
	Valid     bool       `json:"valid"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ============================================================================
// БИЗНЕС-ОШИБКИ (ошибки, которые понимает бизнес-логика)
// ============================================================================

var (
	// ErrInvalidToken — токен неверный или просрочен
	ErrInvalidToken = &BusinessError{Code: "INVALID_TOKEN", Message: "Invalid or expired API token", HTTPStatus: 401}

	// ErrInsufficientBalance — недостаточно средств
	ErrInsufficientBalance = &BusinessError{Code: "INSUFFICIENT_BALANCE", Message: "Insufficient balance. Please top up.", HTTPStatus: 402}

	// ErrUserNotFound — пользователь не найден
	ErrUserNotFound = &BusinessError{Code: "USER_NOT_FOUND", Message: "User not found", HTTPStatus: 404}

	// ErrModelNotFound — модель не найдена
	ErrModelNotFound = &BusinessError{Code: "MODEL_NOT_FOUND", Message: "Model not found", HTTPStatus: 400}

	// ErrProviderError — ошибка провайдера AI
	ErrProviderError = &BusinessError{Code: "PROVIDER_ERROR", Message: "AI provider error", HTTPStatus: 502}

	// ErrInternal — внутренняя ошибка сервера
	ErrInternal = &BusinessError{Code: "INTERNAL_ERROR", Message: "Internal server error", HTTPStatus: 500}
)

// BusinessError — структура бизнес-ошибки (содержит HTTP статус для Handler)
type BusinessError struct {
	Code       string
	Message    string
	HTTPStatus int
	Err        error // внутренняя ошибка (для логирования)
}

func (e *BusinessError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap — для errors.Is / errors.As
func (e *BusinessError) Unwrap() error {
	return e.Err
}

// NewBusinessError — конструктор бизнес-ошибки
func NewBusinessError(code, message string, httpStatus int, err error) *BusinessError {
	return &BusinessError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}
