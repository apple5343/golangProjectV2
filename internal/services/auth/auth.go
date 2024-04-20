package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/apple5343/golangProjectV2/internal/domain/models"
	"github.com/apple5343/golangProjectV2/internal/lib/jwt"
	storage "github.com/apple5343/golangProjectV2/internal/storage/sqlite"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type DataBase interface {
	AddUser(user models.User) (int64, error)
	CheckLogin(name, passwordHash string) (int64, error)
	IsAdmin(userID int) (bool, error)
	IsExist(name string) (models.User, error)
	GetUserInfo(userID int) (string, error)
}

type Auth struct {
	db       DataBase
	secret   string
	tokenTTL time.Duration
}

func New(db DataBase, secret string, tokenTTL time.Duration) *Auth {
	return &Auth{db: db, secret: secret, tokenTTL: tokenTTL}
}

func (a *Auth) Login(name, password string) (string, error) {
	const op = "Auth.Login"
	user, err := a.db.IsExist(name)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}
	token, err := jwt.NewToken(user, a.secret, a.tokenTTL)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return token, nil
}

func (a *Auth) Register(name, password string) (int64, error) {
	const op = "Auth.Register"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}
	isAdmin := 0
	if name == "admin" {
		isAdmin = 1
	}
	user := models.User{Name: name, PasswordHash: string(passwordHash), IsAdmin: isAdmin}
	id, err := a.db.AddUser(user)
	if err != nil {
		if err == storage.ErrUserExists {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}
		return 0, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}
	return id, nil
}

func (a *Auth) IsAdmin(userID int64) (bool, error) {
	isAdmin, err := a.db.IsAdmin(int(userID))
	return isAdmin, err
}

func (a *Auth) GetUserInfo(userID int64) (string, error) {
	userInfo, err := a.db.GetUserInfo(int(userID))
	return userInfo, err
}
