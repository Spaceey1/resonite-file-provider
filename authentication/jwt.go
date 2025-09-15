package authentication

import (
	"os"
	"time"
	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = getJWTKey()

func getJWTKey() []byte {
	if key := os.Getenv("JWT_SECRET_KEY"); key != "" {
        return []byte(key)
    } else if info, err := os.Stat("/run/secrets/jwt.key"); err == nil && !info.IsDir() {
		file, err := os.ReadFile("/run/secrets/jwt.key")
		if err != nil {
			panic("error while reading jwt secret")
		}
		return file
	}
    panic("JWT_SECRET couldn't be found")
}

type Claims struct {
    Username string `json:"username"`
    UID int `json:"uid"`
    jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for a username
func GenerateToken(username string, uId int) (string, error) {
    claims := &Claims{
        Username: username,
	UID: uId,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(730 * time.Hour)),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtKey)
}

// ParseToken validates and extracts claims from a JWT
func ParseToken(tokenStr string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return jwtKey, nil
    })
    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, jwt.ErrTokenSignatureInvalid
}

