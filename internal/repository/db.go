package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB — обёртка над пулом соединений
type DB struct {
	pool Pooler
}

// NewDB — создаёт новое подключение к БД (конструктор!)
func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	log.Println("Подключение к базе данных...")

	// Парсим конфигурацию
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Настраиваем пул соединений
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnIdleTime = 5 * time.Minute

	// Создаём пул
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("База данных подключена успешно")
	return &DB{pool: pool}, nil
}

// Close — закрывает подключение к БД
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
		log.Println("Соединение с БД закрыто")
	}
}

// Ping — проверяет соединение с БД
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

/*Что важно здесь:

Элемент	Зачем
pgxpool.Pool	Пул соединений (эффективно для высоких нагрузок)
ParseConfig	Позволяет настроить пул (MaxConns, MinConns)
Ping()	Проверяет, что БД действительно доступна
Close()	Освобождает ресурсы при завершении
*/
