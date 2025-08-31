package auth

import "fmt"

type AuthError struct {
	Msg string
	err error
}

func Error(format string, a ...any) error {
	return &AuthError{
		Msg: fmt.Sprintf(format, a...),
	}
}

func (e *AuthError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("[AUTH] %s (previous: %v)", e.Msg, e.err)
	}
	return fmt.Sprintf("[AUTH] %s", e.Msg)
}

func (e *AuthError) Unwrap() error {
	return e.err
}

var (
	ErrUnauthorized       = &AuthError{Msg: "unauthorized"}
	ErrTokenMissing       = &AuthError{Msg: "token is missing"}
	ErrTokenExpired       = &AuthError{Msg: "token is expired"}
	ErrTokenInvalid       = &AuthError{Msg: "token is invalid"}
	ErrUserNotFound       = &AuthError{Msg: "user not found"}
	ErrInvalidCredentials = &AuthError{Msg: "invalid credentials"}
)
