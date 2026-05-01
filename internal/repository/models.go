package repository

import (
	"time"
)

// UserDB — модель пользователя в БД (Database DTO)
type UserDB struct {
	ID              string     // UUID из БД
	UserID          string     // логин пользователя (уникальный)
	BalanceUSD      float64    // баланс в долларах
	TotalTokensUsed int64      // сколько токенов потрачено всего
	CreatedAt       time.Time  // дата создания
	UpdatedAt       time.Time  // дата обновления
	DeletedAt       *time.Time // мягкое удаление (nil = не удалён)
}

// TokenDB — модель API токена в БД
type TokenDB struct {
	ID        int        // первичный ключ
	Token     string     // сам токен (sk-xxxxx)
	UserID    string     // владелец токена
	Name      string     // человекочитаемое имя
	IsActive  bool       // активен ли токен
	CreatedAt time.Time  // дата создания
	ExpiresAt *time.Time // дата истечения (nil = бессрочный)
}

/*Обратите внимание:
В UserDB есть поля CreatedAt, UpdatedAt, DeletedAt — они нужны только для БД, клиент их не видит.

В TokenDB есть поле Token — это секрет, он должен быть только в репозитории, не должен попадать в HTTP ответ.
*/
