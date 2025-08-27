package user

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"sufirmart/internal/domain"
	"sufirmart/internal/security"
	"sync"
)

type UserService struct {
	db     *sql.DB
	logger *zap.Logger
	mutex  sync.Mutex
}

func NewUserService(db *sql.DB, logger *zap.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

var ErrLoginAlreadyExists = errors.New("login already exists")

func (us *UserService) RegisterUser(username string, password string) error {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	if username == "" || password == "" {
		return errors.New("username and password are required")
	}

	ctx := context.Background()
	builder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	alreadyExists := us.assertUserExists(username, builder, ctx)
	if alreadyExists != nil {
		return alreadyExists
	}

	u, err := uuid.NewV7()
	if err != nil {
		return err
	}
	uid, err := domain.NewUserID(u.String())
	if err != nil {
		return err
	}

	hashed, err := security.PasswordHash(password)
	if err != nil {
		return err
	}

	insSQL, insArgs, err := builder.
		Insert(`"sufirmart"."user"`).
		Columns("id", "login", "password").
		Values(uid, username, hashed).
		ToSql()
	if err != nil {
		return err
	}

	if _, err = us.db.ExecContext(ctx, insSQL, insArgs...); err != nil {
		return err
	}

	return nil
}

func (us *UserService) assertUserExists(username string, builder squirrel.StatementBuilderType, ctx context.Context) error {
	checkSQL, checkArgs, err := builder.
		Select("TRUE").
		From(`"sufirmart"."user"`).
		Where(squirrel.Eq{"login": username}).
		ToSql()
	if err != nil {
		return err
	}

	var userExists bool
	err = us.db.QueryRowContext(ctx, checkSQL, checkArgs...).Scan(&userExists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && userExists {
		return ErrLoginAlreadyExists
	}

	return nil
}
