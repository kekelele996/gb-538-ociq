package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"industrial-noise-source-attribution/backend/internal/model"
	"industrial-noise-source-attribution/backend/internal/repository"
	"industrial-noise-source-attribution/backend/internal/util"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required,min=2,max=64"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

type UserResponse struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type LoginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

type AuditLogResponse struct {
	ID         uint      `json:"id"`
	RequestID  string    `json:"request_id"`
	ActorID    uint      `json:"actor_id"`
	ActorName  string    `json:"actor_name"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uint      `json:"entity_id"`
	Before     any       `json:"before"`
	After      any       `json:"after"`
	Metadata   any       `json:"metadata"`
	CreatedAt  time.Time `json:"created_at"`
}

type AccessService struct {
	repository *repository.SupportRepository
	jwtSecret  []byte
	jwtExpiry  time.Duration
}

func NewAccessService(repo *repository.SupportRepository, secret string, expiry time.Duration) *AccessService {
	return &AccessService{repository: repo, jwtSecret: []byte(secret), jwtExpiry: expiry}
}

func (s *AccessService) Login(ctx context.Context, request LoginRequest) (LoginResponse, error) {
	user, err := s.repository.FindUserByUsername(ctx, request.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) != nil {
		return LoginResponse{}, util.NewError(http.StatusUnauthorized, "invalid_credentials", "用户名或密码不正确", nil)
	}
	expiresAt := time.Now().UTC().Add(s.jwtExpiry)
	claims := jwt.MapClaims{
		"sub": fmt.Sprint(user.ID), "username": user.Username, "display_name": user.DisplayName,
		"role": user.Role, "exp": expiresAt.Unix(), "iat": time.Now().UTC().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("sign access token: %w", err)
	}
	return LoginResponse{
		Token: signed, ExpiresAt: expiresAt,
		User: UserResponse{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role},
	}, nil
}

func (s *AccessService) ListAudits(ctx context.Context, filter repository.AuditFilter) ([]AuditLogResponse, error) {
	logs, err := s.repository.ListAudits(ctx, filter)
	if err != nil {
		return nil, err
	}
	responses := make([]AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		responses = append(responses, AuditLogResponse{
			ID: log.ID, RequestID: log.RequestID, ActorID: log.ActorID, ActorName: log.ActorName,
			Action: log.Action, EntityType: log.EntityType, EntityID: log.EntityID,
			Before: decodeJSONAny(log.BeforeJSON), After: decodeJSONAny(log.AfterJSON),
			Metadata: decodeJSONAny(log.MetadataJSON), CreatedAt: log.CreatedAt,
		})
	}
	return responses, nil
}

func newAudit(actor model.Actor, action, entityType string, entityID uint, before, after, metadata any) *model.AuditLog {
	return &model.AuditLog{
		RequestID: actor.RequestID, ActorID: actor.ID, ActorName: actor.DisplayName,
		Action: action, EntityType: entityType, EntityID: entityID,
		BeforeJSON: mustJSON(before), AfterJSON: mustJSON(after), MetadataJSON: mustJSON(metadata),
		CreatedAt: time.Now().UTC(),
	}
}

func mustJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func decodeJSONAny(raw string) any {
	if raw == "" {
		return map[string]any{}
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	return value
}

func decodeSpectrum(raw string) (map[string]float64, error) {
	value := make(map[string]float64)
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("decode stored spectrum: %w", err)
	}
	return value, nil
}

func mapRepositoryError(err error, notFoundMessage, conflictMessage string) error {
	if err == nil {
		return nil
	}
	if repository.IsNotFound(err) || errors.Is(err, gorm.ErrRecordNotFound) {
		return util.NotFound(notFoundMessage)
	}
	if repository.IsConflict(err) || errors.Is(err, gorm.ErrDuplicatedKey) {
		return util.Conflict(conflictMessage, err)
	}
	return err
}
