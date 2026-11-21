package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthSessionRepository interface {
	Create(
		ctx context.Context,
		userID int64,
		tokenHash string,
		expiresAt time.Time,
	) error

	FindByTokenHash(
		ctx context.Context,
		tokenHash string,
	) (int64, time.Time, error)

	Revoke(
		ctx context.Context,
		tokenHash string,
	) error
}

type authSessionRepository struct {
	db *pgxpool.Pool
}

func NewAuthSessionRepository(
	db *pgxpool.Pool,
) AuthSessionRepository {
	return &authSessionRepository{
		db: db,
	}
}

// Create stores a new refresh token session.
func (r *authSessionRepository) Create(
	ctx context.Context,
	userID int64,
	tokenHash string,
	expiresAt time.Time,
) error {
	query := `
		INSERT INTO auth_sessions (
			user_id,
			refresh_token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		userID,
		tokenHash,
		expiresAt,
	)

	return err
}

// FindByTokenHash retrieves an active refresh token session.
func (r *authSessionRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (int64, time.Time, error) {
	query := `
		SELECT user_id, expires_at
		FROM auth_sessions
		WHERE refresh_token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > CURRENT_TIMESTAMP
	`

	var userID int64
	var expiresAt time.Time

	err := r.db.QueryRow(
		ctx,
		query,
		tokenHash,
	).Scan(
		&userID,
		&expiresAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, time.Time{}, pgx.ErrNoRows
		}

		return 0, time.Time{}, err
	}

	return userID, expiresAt, nil
}

// Revoke invalidates a refresh token session.
func (r *authSessionRepository) Revoke(
	ctx context.Context,
	tokenHash string,
) error {
	query := `
		UPDATE auth_sessions
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE refresh_token_hash = $1
		  AND revoked_at IS NULL
	`

	result, err := r.db.Exec(
		ctx,
		query,
		tokenHash,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
