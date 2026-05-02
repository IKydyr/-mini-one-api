package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"mini_one_api/internal/service"
)

// ChatHandler — обработчик чат-запросов
type ChatHandler struct {
	chatService service.ChatService
	authService service.AuthService
	logger      *slog.Logger
}

// NewChatHandler — конструктор
func NewChatHandler(chatService service.ChatService, authService service.AuthService, logger *slog.Logger) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		authService: authService,
		logger:      logger,
	}
}

// HandleChatCompletion — POST /v1/chat/completions
func (h *ChatHandler) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	// 1. Проверяем метод
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST allowed")
		return
	}

	// 2. Извлекаем токен
	token := r.Header.Get("Authorization")
	if token == "" {
		h.sendError(w, http.StatusUnauthorized, "missing_token", "Authorization header is required")
		return
	}
	// Убираем "Bearer " если есть
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	userID, err := h.authService.Authenticate(r.Context(), token)
	if err != nil {
		h.sendError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token")
		return
	}

	// 3. Парсим JSON запрос
	var httpReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&httpReq); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	// 4. Быстрая валидация
	if httpReq.Model == "" {
		h.sendError(w, http.StatusBadRequest, "missing_field", "model is required")
		return
	}
	if len(httpReq.Messages) == 0 {
		h.sendError(w, http.StatusBadRequest, "missing_field", "messages are required")
		return
	}

	// 5. МАППИНГ: HTTP DTO → Service DTO
	serviceReq := service.ChatRequest{
		UserID:   userID,
		Model:    httpReq.Model,
		Stream:   httpReq.Stream,
		Messages: make([]service.ChatMessage, len(httpReq.Messages)),
	}
	for i, msg := range httpReq.Messages {
		serviceReq.Messages[i] = service.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// 6. Вызываем сервис — обычный или стриминговый
	if httpReq.Stream {
		h.handleStream(w, r, serviceReq)
		return
	}

	result, err := h.chatService.ProcessChat(r.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Chat processing failed", "error", err)

		if bizErr, ok := err.(*service.BusinessError); ok {
			h.sendError(w, bizErr.HTTPStatus, bizErr.Code, bizErr.Message)
			return
		}
		h.sendError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// 7. МАППИНГ: Service DTO → HTTP DTO
	httpResp := h.buildChatResponse(result, httpReq.Model)

	// 8. Отправляем ответ
	h.sendJSON(w, http.StatusOK, httpResp)
}

// buildChatResponse — строит HTTP ответ из данных сервиса
func (h *ChatHandler) buildChatResponse(result *service.ChatResponse, model string) ChatResponse {
	resp := ChatResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
	}
	resp.Choices = append(resp.Choices, struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}{
		Index: 0,
		Message: struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{Role: "assistant", Content: result.Content},
		FinishReason: "stop",
	})
	resp.Usage.PromptTokens = result.TokensUsed / 2
	resp.Usage.CompletionTokens = result.TokensUsed / 2
	resp.Usage.TotalTokens = result.TokensUsed

	return resp
}

// sendJSON — отправляет JSON ответ
func (h *ChatHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON", "error", err)
	}
}

// sendError — отправляет ошибку
func (h *ChatHandler) sendError(w http.ResponseWriter, status int, errType, message string) {
	resp := ErrorResponse{}
	resp.Error.Message = message
	resp.Error.Type = errType
	resp.Error.Code = http.StatusText(status)
	h.sendJSON(w, status, resp)
}

func (h *ChatHandler) handleStream(w http.ResponseWriter, r *http.Request, req service.ChatRequest) {
	// Устанавливаем SSE заголовки
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	messages, errChan := h.chatService.ProcessChatStream(r.Context(), req)

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.sendError(w, http.StatusInternalServerError, "streaming_unsupported", "Streaming not supported")
		return
	}

	for {
		select {
		case chunk, open := <-messages:
			if !open {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		case err := <-errChan:
			if err != nil {
				fmt.Fprintf(w, "data: [ERROR] %s\n\n", err.Error())
				flusher.Flush()
			}
			return
		case <-r.Context().Done():
			return
		}
	}
}
