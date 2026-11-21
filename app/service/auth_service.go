package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/saniaaww/skillprof/app/model"
	"github.com/saniaaww/skillprof/app/repository"
	"github.com/saniaaww/skillprof/config"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrJWTSecretMissing    = errors.New("JWT secret is not configured")
	ErrWeakPassword        = errors.New("password must be at least 8 characters")
	ErrInvalidToken        = errors.New("invalid or expired access token")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

type AuthService interface {
	Register(ctx context.Context, req model.RegisterRequest) (*model.User, error)
	Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, req model.RefreshTokenRequest) (*model.TokenResponse, error)
	Me(ctx context.Context, userID int64) (*model.User, error)
	VerifyToken(ctx context.Context, token string) (*model.VerifyTokenResponse, error)
}

type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.AuthSessionRepository
	cfg         config.AppConfig
}

func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.AuthSessionRepository,
	cfg config.AppConfig,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		cfg:         cfg,
	}
}

// Register creates a new user account.
func (s *authService) Register(
	ctx context.Context,
	req model.RegisterRequest,
) (*model.User, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, ErrInvalidInput
	}

	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, ErrInvalidInput
	}

	if len(req.Password) < 8 {
		return nil, ErrWeakPassword
	}

	existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	createdUser.Password = ""
	return createdUser, nil
}

// Login authenticates a user and creates an access-refresh token pair.
func (s *authService) Login(
	ctx context.Context,
	req model.LoginRequest,
) (*model.AuthResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" || req.Password == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, refreshToken, expiresIn, err := s.createTokenPair(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	user.Password = ""

	return &model.AuthResponse{
		User:         *user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
	}, nil
}

// Logout revokes a refresh-token session.
func (s *authService) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return ErrInvalidRefreshToken
	}

	hash := hashToken(refreshToken)

	_, _, err := s.sessionRepo.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidRefreshToken
		}
		return err
	}

	return s.sessionRepo.Revoke(ctx, hash)
}

// RefreshToken rotates the refresh token and creates a new access token.
func (s *authService) RefreshToken(
	ctx context.Context,
	req model.RefreshTokenRequest,
) (*model.TokenResponse, error) {
	oldToken := strings.TrimSpace(req.RefreshToken)
	if oldToken == "" {
		return nil, ErrInvalidRefreshToken
	}

	oldHash := hashToken(oldToken)

	userID, _, err := s.sessionRepo.FindByTokenHash(ctx, oldHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	// Revoke the old refresh token so it cannot be reused.
	if err := s.sessionRepo.Revoke(ctx, oldHash); err != nil {
		return nil, err
	}

	accessToken, refreshToken, expiresIn, err := s.createTokenPair(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
	}, nil
}

// Me retrieves the authenticated user's profile.
func (s *authService) Me(
	ctx context.Context,
	userID int64,
) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.Password = ""
	return user, nil
}

// VerifyToken validates an access JWT.
func (s *authService) VerifyToken(
	ctx context.Context,
	rawToken string,
) (*model.VerifyTokenResponse, error) {
	_ = ctx

	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			if s.cfg.JWTSecret == "" {
				return nil, ErrJWTSecretMissing
			}

			return []byte(s.cfg.JWTSecret), nil
		},
	)

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	userIDValue, ok := claims["user_id"].(float64)
	if !ok || userIDValue <= 0 {
		return nil, ErrInvalidToken
	}

	return &model.VerifyTokenResponse{
		Valid:  true,
		UserID: int64(userIDValue),
	}, nil
}

// createTokenPair generates an access JWT and an opaque refresh token.
func (s *authService) createTokenPair(
	ctx context.Context,
	userID int64,
) (string, string, int64, error) {
	if s.cfg.JWTSecret == "" {
		return "", "", 0, ErrJWTSecretMissing
	}

	accessHours := s.cfg.JWTExpiresIn
	if accessHours <= 0 {
		accessHours = 1
	}

	refreshHours := s.cfg.JWTRefreshExpiresIn
	if refreshHours <= 0 {
		refreshHours = 168
	}

	now := time.Now()
	accessExpiresAt := now.Add(time.Duration(accessHours) * time.Hour)

	claims := jwt.MapClaims{
		"user_id": userID,
		"iat":     now.Unix(),
		"exp":     accessExpiresAt.Unix(),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	accessToken, err := jwtToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", 0, err
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", 0, err
	}

	refreshToken := base64.RawURLEncoding.EncodeToString(randomBytes)
	refreshExpiresAt := now.Add(time.Duration(refreshHours) * time.Hour)

	if err := s.sessionRepo.Create(
		ctx,
		userID,
		hashToken(refreshToken),
		refreshExpiresAt,
	); err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, int64(time.Duration(accessHours) * 3600), nil
}

// hashToken hashes a refresh token before storing it.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
