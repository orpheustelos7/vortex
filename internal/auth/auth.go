package auth

import (
"errors"
"fmt"
"net/http"
"slices"
"strings"

"github.com/golang-jwt/jwt/v5"
"github.com/orpheustelos7/vortex/internal/config"
)

func ValidateRequest(r *http.Request, policy config.Policy) error {
if len(policy.APIKeys) == 0 && policy.JWTSecret == "" {
return nil
}

apiKey := r.Header.Get("X-API-Key")
if apiKey != "" && slices.Contains(policy.APIKeys, apiKey) {
return nil
}

authHeader := r.Header.Get("Authorization")
if policy.JWTSecret != "" && strings.HasPrefix(authHeader, "Bearer ") {
tokenString := strings.TrimPrefix(authHeader, "Bearer ")
token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
return nil, fmt.Errorf("unexpected signing method")
}
return []byte(policy.JWTSecret), nil
})
if err == nil && token.Valid {
return nil
}
}

return errors.New("unauthorized")
}
