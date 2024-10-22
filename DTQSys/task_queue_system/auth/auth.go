package auth

import (
    "database/sql"
    "errors"
    "os"
    "time"
    "task_queue_system/models"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET_KEY"))

type Claims struct {
    Username string `json:"username"`
    jwt.RegisteredClaims
}

func RegisterUser(db *sql.DB, username, password string) error {
    // Check if the user already exists
    var exists bool
    err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)", username).Scan(&exists)
    if err != nil {
        return err
    }
    if exists {
        return errors.New("user already exists")
    }

    // Hash the password
    passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }

    // Insert the new user
    _, err = db.Exec("INSERT INTO users (username, password_hash) VALUES ($1, $2)", username, string(passwordHash))
    return err
}

func AuthenticateUser(db *sql.DB, username, password string) (string, string, error) {
    var passwordHash string
    err := db.QueryRow("SELECT password_hash FROM users WHERE username=$1", username).Scan(&passwordHash)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", "", errors.New("user not found")
        }
        return "", "", err
    }

    // Compare the provided password with the stored hash
    err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
    if err != nil {
        return "", "", errors.New("invalid credentials")
    }

    // Generate tokens
    accessToken, refreshToken, err := GenerateTokens(username)
    if err != nil {
        return "", "", err
    }

    return accessToken, refreshToken, nil
}

func GenerateTokens(username string) (accessToken string, refreshToken string, err error) {
    accessTokenExp := time.Now().Add(15 * time.Minute) // Short-lived access token
    refreshTokenExp := time.Now().Add(7 * 24 * time.Hour) // Long-lived refresh token

    accessClaims := &Claims{
        Username: username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(accessTokenExp),
        },
    }

    refreshClaims := &Claims{
        Username: username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(refreshTokenExp),
        },
    }

    at := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    rt := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

    accessToken, err = at.SignedString(jwtKey)
    if err != nil {
        return "", "", err
    }

    refreshToken, err = rt.SignedString(jwtKey)
    if err != nil {
        return "", "", err
    }

    return accessToken, refreshToken, nil
}

func ValidateToken(tokenStr string) (*Claims, error) {
    claims := &Claims{}
    token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
        return jwtKey, nil
    })
    if err != nil {
        return nil, err
    }
    if !token.Valid {
        return nil, errors.New("invalid token")
    }
    return claims, nil
}
