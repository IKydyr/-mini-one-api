package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"mini_one_api/internal/service"
)

// UserHandler — обработчик запросов пользователя
type UserHandler struct {
	userService service.UserService
	authService service.AuthService
	logger      *slog.Logger
}

// NewUserHandler — конструктор
func NewUserHandler(userService service.UserService, authService service.AuthService, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		authService: authService,
		logger:      logger,
	}
}

// GetUserInfo — GET /v1/user/info
func (h *UserHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	// 1. Извлекаем токен из заголовка
	token := r.Header.Get("Authorization")
	if token == "" {
		h.sendError(w, http.StatusUnauthorized, "missing_token", "Authorization header is required")
		return
	}

	// Убираем "Bearer " если есть
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// 2. В реальном коде здесь была бы аутентификация
	// Для простоты используем заглушку: userID = "test_user_1" теперь заглушку убрал
	userID, err := h.authService.Authenticate(r.Context(), token)
	if err != nil {
		h.sendError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token")
		return
	}

	// 3. Вызываем сервис
	resp, err := h.userService.GetUserInfo(r.Context(), service.GetUserInfoRequest{
		UserID: userID,
	})
	if err != nil {
		h.logger.Error("Failed to get user info", "error", err)

		// Превращаем бизнес-ошибку в HTTP ответ
		if bizErr, ok := err.(*service.BusinessError); ok {
			h.sendError(w, bizErr.HTTPStatus, bizErr.Code, bizErr.Message)
			return
		}
		h.sendError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
		return
	}

	// 4. Формируем HTTP ответ
	response := GetUserResponse{
		UserID:      resp.UserID,
		BalanceUSD:  resp.BalanceUSD,
		TotalTokens: resp.TotalTokens,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// sendJSON — отправляет JSON ответ
func (h *UserHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON", "error", err)
	}
}

// sendError — отправляет ошибку в формате JSON
func (h *UserHandler) sendError(w http.ResponseWriter, status int, errType, message string) {
	resp := ErrorResponse{}
	resp.Error.Message = message
	resp.Error.Type = errType
	resp.Error.Code = http.StatusText(status)
	h.sendJSON(w, status, resp)
}
