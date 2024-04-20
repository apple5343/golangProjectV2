package tests

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/apple5343/golangProjectV2/tests/test"
	s "github.com/apple5343/grpc"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	passwordLen = 10
)

func TestRegisterLogin_Login_HappyPath(t *testing.T) {
	ctx, st := test.New(t)

	name := gofakeit.Email()
	pass := randomPassword()

	respReg, err := st.AuthClient.Register(ctx, &s.RegisterRequest{
		Name:     name,
		Password: pass,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, respReg.GetUserId())

	respLogin, err := st.AuthClient.Login(ctx, &s.LoginRequest{
		Name:     name,
		Password: pass,
	})
	require.NoError(t, err)

	token := respLogin.GetToken()
	require.NotEmpty(t, token)

	loginTime := time.Now()

	tokenParsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(st.Cfg.SecretJWT), nil
	})
	require.NoError(t, err)

	claims, ok := tokenParsed.Claims.(jwt.MapClaims)
	require.True(t, ok)

	assert.Equal(t, respReg.GetUserId(), int64(claims["id"].(float64)))

	assert.InDelta(t, loginTime.Add(st.Cfg.TokenTTL).Unix(), claims["exp"].(float64), 1)
}

func TestRegisterLogin_DuplicatedRegistration(t *testing.T) {
	ctx, st := test.New(t)

	name := gofakeit.Email()
	pass := randomPassword()

	respReg, err := st.AuthClient.Register(ctx, &s.RegisterRequest{
		Name:     name,
		Password: pass,
	})
	require.NoError(t, err)
	require.NotEmpty(t, respReg.GetUserId())

	respReg, err = st.AuthClient.Register(ctx, &s.RegisterRequest{
		Name:     name,
		Password: pass,
	})
	require.Error(t, err)
	assert.Empty(t, respReg.GetUserId())
	assert.ErrorContains(t, err, "user already exists")
}

func TestRegister_FailCases(t *testing.T) {
	ctx, st := test.New(t)

	tests := []struct {
		name          string
		uname         string
		password      string
		expectedError string
	}{
		{
			name:          "Пустой пароль",
			uname:         gofakeit.Email(),
			password:      "",
			expectedError: "password is required",
		},
		{
			name:          "Пустое имя",
			uname:         "",
			password:      randomPassword(),
			expectedError: "name is required",
		},
		{
			name:          "Пустые данные",
			uname:         "",
			password:      "",
			expectedError: "name is required",
		},
		{
			name:          "Короткий пароль",
			uname:         gofakeit.Email(),
			password:      "123",
			expectedError: "Пароль должен содержать не менее 5 символов",
		},
		{
			name:          "Пароль без цифр",
			uname:         gofakeit.Email(),
			password:      "Aaaaaaaaaaa",
			expectedError: "Пароль должен содердать цифры",
		},
		{
			name:          "Пароль без заглавный букв",
			uname:         gofakeit.Email(),
			password:      "aaaaa1234!",
			expectedError: "Пароль должен содержать загланые буквы",
		},
		{
			name:          "Пароль без строчных букв",
			uname:         gofakeit.Email(),
			password:      "AAAAA1234!",
			expectedError: "Пароль должен содержать строчные буквы",
		},
		{
			name:          "Пароль без спец символов",
			uname:         gofakeit.Email(),
			password:      "Aa1234534",
			expectedError: "Пароль должен содержать специальный символ",
		},
		{
			name:          "Пароль на русском языке",
			uname:         gofakeit.Email(),
			password:      "Password1234!хыхы",
			expectedError: "Пароль должен содержать только английские буквы",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.AuthClient.Register(ctx, &s.RegisterRequest{
				Name:     tt.uname,
				Password: tt.password,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestLogin_FailCases(t *testing.T) {
	ctx, st := test.New(t)

	tests := []struct {
		name          string
		uname         string
		password      string
		expectedError string
	}{
		{
			name:          "Пустой пароль",
			uname:         gofakeit.Email(),
			password:      "",
			expectedError: "password is required",
		},
		{
			name:          "Пустое имя",
			uname:         "",
			password:      randomPassword(),
			expectedError: "name is required",
		},
		{
			name:          "Пустые данные",
			uname:         "",
			password:      "",
			expectedError: "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.AuthClient.Register(ctx, &s.RegisterRequest{
				Name:     gofakeit.Email(),
				Password: randomPassword(),
			})
			require.NoError(t, err)

			_, err = st.AuthClient.Login(ctx, &s.LoginRequest{
				Name:     tt.uname,
				Password: tt.password,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func randomPassword() string {
	return generatePassword(2, 2, 2, 2)
}

func generatePassword(minUpperCase, minLowerCase, minDigits, minSpecials int) string {
	const (
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		digitChars   = "0123456789"
		specialChars = "!@#$%^&*"
	)
	var password strings.Builder
	password.Grow(passwordLen)

	for _, requiredChars := range []struct {
		chars  string
		amount int
	}{
		{upperChars, minUpperCase},
		{lowerChars, minLowerCase},
		{digitChars, minDigits},
		{specialChars, minSpecials},
	} {
		for i := 0; i < requiredChars.amount; i++ {
			randomChar := getRandomChar(requiredChars.chars)
			password.WriteByte(randomChar)
		}
	}

	remainingLength := passwordLen - minUpperCase - minLowerCase - minDigits - minSpecials
	allChars := upperChars + lowerChars + digitChars + specialChars
	for i := 0; i < remainingLength; i++ {
		randomChar := getRandomChar(allChars)
		password.WriteByte(randomChar)
	}
	shuffledPassword := shuffleString(password.String())

	return shuffledPassword
}

func getRandomChar(chars string) byte {
	byteSlice := make([]byte, 1)
	if _, err := rand.Read(byteSlice); err != nil {
		return 0
	}
	index := byteSlice[0] % byte(len(chars))
	return chars[index]
}

func shuffleString(s string) string {
	r := []rune(s)
	for i := len(r) - 1; i > 0; i-- {
		j := getRandomInt(i + 1)
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func getRandomInt(n int) int {
	byteSlice := make([]byte, 1)
	for {
		if _, err := rand.Read(byteSlice); err != nil {
			return 0
		}
		if int(byteSlice[0]) < n*256/n {
			return int(byteSlice[0]) % n
		}
	}
}
