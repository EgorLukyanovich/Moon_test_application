package routing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/egor_lukyanovich/moon_test_application/internal/db"
	json_resp "github.com/egor_lukyanovich/moon_test_application/pkg/json"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const userIDKey contextKey = "userID"

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func getJWTKey() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("test_secret_key_123")
	}
	return []byte(secret)
}

func GetUserIDHelper(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

func CheckTeamRole(ctx context.Context, q *db.Queries, teamID, userID int64, allowedRoles ...string) bool {
	role, err := q.GetUserRoleInTeam(ctx, db.GetUserRoleInTeamParams{
		TeamID: teamID,
		UserID: userID,
	})
	if err != nil {
		return false
	}

	if len(allowedRoles) == 0 {
		return true
	}

	userRoleStr := string(role)
	for _, allowed := range allowedRoles {
		if userRoleStr == allowed {
			return true
		}
	}

	return false
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			json_resp.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing Authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			json_resp.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid Authorization header format")
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return getJWTKey(), nil
		})

		if err != nil || !token.Valid {
			json_resp.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			json_resp.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token claims")
			return
		}

		userIDFloat, ok := claims["sub"].(float64)
		if !ok {
			json_resp.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID in token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, int64(userIDFloat))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type AuthHandlers struct {
	q *db.Queries
}

func NewAuthHandlers(q *db.Queries) *AuthHandlers {
	return &AuthHandlers{q: q}
}

func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json_resp.RespondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		json_resp.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		return
	}

	res, err := h.q.CreateUser(r.Context(), db.CreateUserParams{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	})
	if err != nil {
		json_resp.RespondError(w, http.StatusConflict, "CONFLICT", "user with this email already exists")
		return
	}

	userID, _ := res.LastInsertId()

	json_resp.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      userID,
		"message": "user registered successfully",
	})
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json_resp.RespondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}

	user, err := h.q.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		json_resp.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		json_resp.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(72 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString(getJWTKey())
	if err != nil {
		json_resp.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
		return
	}

	json_resp.RespondJSON(w, http.StatusOK, map[string]string{
		"token": tokenString,
	})
}
