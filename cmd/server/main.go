package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mini_one_api/internal/handler"
	"mini_one_api/internal/provider"
	"mini_one_api/internal/repository"
	"mini_one_api/internal/service"
)

func main() {
	ctx := context.Background()

	// =========================================================================
	// 1. ЗАГРУЗКА КОНФИГУРАЦИИ
	// =========================================================================
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://oneapi:oneapi123@localhost:5432/oneapi?sslmode=disable"
	}

	deepseekAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	if deepseekAPIKey == "" {
		log.Println("WARNING: DEEPSEEK_API_KEY not set, using dummy key")
		deepseekAPIKey = "dummy-key"
	}

	// =========================================================================
	// 2. ИНИЦИАЛИЗАЦИЯ ЛОГГЕРА
	// =========================================================================
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// =========================================================================
	// 3. ПОДКЛЮЧЕНИЕ К БАЗЕ ДАННЫХ (РЕПОЗИТОРИЙ)
	// =========================================================================
	log.Println("Подключение к базе данных...")

	db, err := repository.NewDB(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("База данных подключена успешно")

	// =========================================================================
	// 4. ИНИЦИАЛИЗАЦИЯ РЕПОЗИТОРИЕВ
	// =========================================================================
	userRepo := repository.NewUserRepository(db)
	chargeRepo := repository.NewChargeRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	// =========================================================================
	// 5. ИНИЦИАЛИЗАЦИЯ ПРОВАЙДЕРОВ
	// =========================================================================
	deepseekProvider := provider.NewDeepSeekProvider(deepseekAPIKey)

	// =========================================================================
	// 6. ИНИЦИАЛИЗАЦИЯ СЕРВИСОВ (БИЗНЕС-ЛОГИКА)
	// =========================================================================
	chatService := service.NewChatService(
		userRepo,
		chargeRepo,
		deepseekProvider,
		logger,
	)

	userService := service.NewUserService(
		userRepo,
		tokenRepo,
		logger,
	)

	// =========================================================================
	// 7. ИНИЦИАЛИЗАЦИЯ HANDLER (HTTP)
	// =========================================================================
	chatHandler := handler.NewChatHandler(chatService, logger)
	userHandler := handler.NewUserHandler(userService, logger)

	// =========================================================================
	// 8. НАСТРОЙКА HTTP РОУТЕРА
	// =========================================================================
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/user/info", userHandler.GetUserInfo)
	mux.HandleFunc("POST /v1/chat/completions", chatHandler.HandleChatCompletion)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// =========================================================================
	// 9. ЗАПУСК СЕРВЕРА
	// =========================================================================
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	go func() {
		log.Println("Сервер запущен на http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// =========================================================================
	// 10. GRACEFUL SHUTDOWN
	// =========================================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Остановка сервера...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Сервер остановлен")
}
