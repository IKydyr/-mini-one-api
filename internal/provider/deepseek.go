package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// КОНСТАНТЫ
// ============================================================================

const (
	DeepSeekAPIURL = "https://api.deepseek.com/v1/chat/completions"
	DefaultTimeout = 60 * time.Second
)

// ============================================================================
// DTO ДЛЯ ЗАПРОСОВ/ОТВЕТОВ DEEPSEEK
// ============================================================================

type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DeepSeekRequest struct {
	Model       string            `json:"model"`
	Messages    []DeepSeekMessage `json:"messages"`
	Stream      bool              `json:"stream,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
}

type DeepSeekResponse struct {
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

// ============================================================================
// DEEPSEEK STREAM (ваше правильное решение!)
// ============================================================================
// DeepSeekStreamChunk — полная структура одного чанка потокового ответа
type DeepSeekStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// DeepSeekStream — управляемый поток для SSE (Server-Sent Events)
type DeepSeekStream struct {
	Ch     <-chan string      // канал с чанками текста (публичный, для range)
	cancel context.CancelFunc // для закрытия потока
}

// Close — закрывает поток и освобождает ресурсы
func (s *DeepSeekStream) Close() {
	s.cancel()
}

// ============================================================================
// ПРОВАЙДЕР
// ============================================================================

type DeepSeekProvider struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

func NewDeepSeekProvider(apiKey string) *DeepSeekProvider {
	return &DeepSeekProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL: DeepSeekAPIURL,
	}
}

// ChatCompletion — обычный (не потоковый) запрос
func (p *DeepSeekProvider) ChatCompletion(ctx context.Context, req DeepSeekRequest) (*DeepSeekResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var deepseekResp DeepSeekResponse
	if err := json.Unmarshal(body, &deepseekResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &deepseekResp, nil
}

// ChatCompletionStream — потоковый запрос (возвращает DeepSeekStream с каналом)
func (p *DeepSeekProvider) ChatCompletionStream(ctx context.Context, req DeepSeekRequest) (*DeepSeekStream, error) {
	req.Stream = true

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Создаём контекст для управления потоком
	streamCtx, cancel := context.WithCancel(ctx)

	// Канал для чанков (буфер 10 для производительности)
	chunks := make(chan string, 10)

	// Запускаем горутину для чтения SSE
	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)

		for scanner.Scan() {
			select {
			case <-streamCtx.Done():
				return
			default:
			}

			line := scanner.Text()

			// Пропускаем пустые строки
			if line == "" {
				continue
			}

			// Убираем "data: " префикс
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// Проверяем конец потока
			if data == "[DONE]" {
				return
			}

			// Отправляем чанк в канал
			var chunk DeepSeekStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				select {
				case chunks <- chunk.Choices[0].Delta.Content:
				case <-streamCtx.Done():
					return
				}
			}

		}

		if err := scanner.Err(); err != nil {
			// Можно логировать, но не прерываем
			_ = err
		}
	}()

	return &DeepSeekStream{
		Ch:     chunks,
		cancel: cancel,
	}, nil
}
