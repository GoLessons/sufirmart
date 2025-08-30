package auth

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"
	"sufirmart/internal/security"
)

type Authentication interface {
	Authenticate(username string, password string) (token string, err error)
	IsAuthenticated(token string) (bool, error)
}

type AuthService struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewAuthService(db *sql.DB, logger *zap.Logger) *AuthService {
	return &AuthService{db: db, logger: logger}
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func (a *AuthService) Authenticate(username string, password string) (string, error) {
	if username == "" || password == "" {
		return "", ErrInvalidCredentials
	}

	ctx := context.Background()
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Ищем пользователя по логину
	var userID, passwordHash string
	selSQL, selArgs, err := sb.
		Select("id", "password").
		From(`"sufirmart"."user"`).
		Where(squirrel.Eq{"login": username}).
		ToSql()
	if err != nil {
		return "", err
	}
	if err = a.db.QueryRowContext(ctx, selSQL, selArgs...).Scan(&userID, &passwordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	// Проверяем пароль
	if !security.PasswordVerify(password, passwordHash) {
		return "", ErrInvalidCredentials
	}

	// Генерируем токен и сохраняем
	token, err := security.GenerateToken(32)
	if err != nil {
		return "", err
	}
	insSQL, insArgs, err := sb.
		Insert(`"sufirmart"."auth"`).
		Columns("user_id", "token").
		Values(userID, token).
		ToSql()
	if err != nil {
		return "", err
	}
	if _, err = a.db.ExecContext(ctx, insSQL, insArgs...); err != nil {
		return "", err
	}

	return token, nil
}

func (a *AuthService) IsAuthenticated(token string) (bool, error) {
	if token == "" {
		return false, nil
	}

	ctx := context.Background()
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	var probe int
	selSQL, selArgs, err := sb.
		Select("1").
		From(`"sufirmart"."auth"`).
		Where(squirrel.And{
			squirrel.Eq{"token": token},
			squirrel.Expr(`"expired_at" > NOW()`),
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, err
	}
	if err = a.db.QueryRowContext(ctx, selSQL, selArgs...).Scan(&probe); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
