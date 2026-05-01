package repository

import "errors"

// Ошибки репозитория (технические, не бизнес-логика)
var (
	// ErrNotFound — запись не найдена в БД
	ErrNotFound = errors.New("record not found")

	// ErrDuplicate — дубликат записи (нарушение уникальности)
	ErrDuplicate = errors.New("duplicate record")

	// ErrDatabase — общая ошибка БД
	ErrDatabase = errors.New("database error")
)

/*
Почему эти ошибки здесь?
Это технические ошибки репозитория.
Они не содержат бизнес-логики (например, "недостаточно средств" — это бизнес-ошибка, она будет в service).
*/
