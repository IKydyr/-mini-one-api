package handler

// ============================================================================
// HTTP DTO ДЛЯ ПОЛЬЗОВАТЕЛЯ
// ============================================================================

// GetUserResponse — ответ с информацией о пользователе
type GetUserResponse struct {
	UserID      string  `json:"user_id"`
	BalanceUSD  float64 `json:"balance_usd"`
	TotalTokens int64   `json:"total_tokens"`
}

// ============================================================================
// HTTP DTO ДЛЯ ЧАТА
// ============================================================================

// ChatRequest — запрос от клиента (совместимо с OpenAI API)
type ChatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream bool `json:"stream,omitempty"`
}

// ChatResponse — ответ клиенту (совместимо с OpenAI API)
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ErrorResponse — ответ с ошибкой
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
