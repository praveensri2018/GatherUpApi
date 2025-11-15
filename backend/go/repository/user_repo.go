/* Place: backend/go/repository/user_repo.go */
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gatherup/models"

	"github.com/google/uuid"
)

type UserRepo struct {
	db *sql.DB
}

type RefreshTokenRow struct {
	ID        string
	UserID    string
	TokenHash string
	Device    *string
	CreatedAt time.Time
	ExpiresAt time.Time
	Revoked   bool
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// CreateUser inserts a new user (profile only) and returns generated id.
func (r *UserRepo) CreateUser(ctx context.Context, u *models.User) (string, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dbo.users (id, mobile_number, mobile_normalized, email, username, created_at, is_deleted)
		VALUES (@p1, @p2, @p3, @p4, @p5, @p6, 0)
	`, id, sqlNullString(u.MobileNumber), sqlNullString(u.MobileNormalized), sqlNullString(u.Email), sqlNullString(u.Username), now)
	if err != nil {
		return "", err
	}
	return id, nil
}

// CreateUserWithPassword - create user row AND credential row in a single transaction.
func (r *UserRepo) CreateUserWithPassword(ctx context.Context, mobileRaw, mobileNorm, passwordHash string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userID := uuid.New().String()
	now := time.Now().UTC()

	// Insert into users (profile only)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO dbo.users (id, mobile_number, mobile_normalized, created_at, is_deleted)
		VALUES (@p1, @p2, @p3, @p4, 0)
	`, userID, mobileRaw, mobileNorm, now)
	if err != nil {
		return "", err
	}

	// Insert into user_credentials
	_, err = tx.ExecContext(ctx, `
		INSERT INTO dbo.user_credentials (user_id, credential_type, credential_identifier, password_hash, created_at, is_deleted)
		VALUES (@p1, @p2, @p3, @p4, @p5, 0)
	`, userID, "password", mobileNorm, passwordHash, now)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return userID, nil
}

// GetByEmailOrMobile returns a user by email or mobile (profile only).
func (r *UserRepo) GetByEmailOrMobile(ctx context.Context, identifier string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, mobile_number, mobile_normalized, email, username, created_at, updated_at, is_deleted
		FROM dbo.users WHERE (email = @p1 OR mobile_number = @p1) AND is_deleted = 0
	`, identifier)
	u := &models.User{}
	var mobile, mobileNorm, email, username sql.NullString
	var updatedAt sql.NullTime
	if err := row.Scan(&u.ID, &mobile, &mobileNorm, &email, &username, &u.CreatedAt, &updatedAt, &u.IsDeleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if mobile.Valid {
		u.MobileNumber = &mobile.String
	}
	if mobileNorm.Valid {
		u.MobileNormalized = &mobileNorm.String
	}
	if email.Valid {
		u.Email = &email.String
	}
	if username.Valid {
		u.Username = &username.String
	}
	if updatedAt.Valid {
		t := updatedAt.Time
		u.UpdatedAt = &t
	}
	return u, nil
}

// GetByID returns profile by user id (no password)
func (r *UserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, mobile_number, mobile_normalized, email, username, created_at, updated_at, is_deleted
		FROM dbo.users WHERE id = @p1 AND is_deleted = 0
	`, id)
	u := &models.User{}
	var mobile, mobileNorm, email, username sql.NullString
	var updatedAt sql.NullTime
	if err := row.Scan(&u.ID, &mobile, &mobileNorm, &email, &username, &u.CreatedAt, &updatedAt, &u.IsDeleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if mobile.Valid {
		u.MobileNumber = &mobile.String
	}
	if mobileNorm.Valid {
		u.MobileNormalized = &mobileNorm.String
	}
	if email.Valid {
		u.Email = &email.String
	}
	if username.Valid {
		u.Username = &username.String
	}
	if updatedAt.Valid {
		t := updatedAt.Time
		u.UpdatedAt = &t
	}
	return u, nil
}

// GetCredentialByIdentifier returns credential row for credential_type='password' and the linked user id.
// Returns (userID, passwordHash, nil) if found, ("","",nil) if not found.
func (r *UserRepo) GetCredentialByIdentifier(ctx context.Context, identifier string) (string, string, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT user_id, password_hash
		FROM dbo.user_credentials
		WHERE credential_type = 'password' AND credential_identifier = @p1 AND is_deleted = 0
	`, identifier)

	var uid sql.NullString
	var ph sql.NullString
	if err := row.Scan(&uid, &ph); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil
		}
		return "", "", err
	}
	userID := ""
	pwHash := ""
	if uid.Valid {
		userID = uid.String
	}
	if ph.Valid {
		pwHash = ph.String
	}
	return userID, pwHash, nil
}

// SaveRefreshToken stores refresh token hash
func (r *UserRepo) SaveRefreshToken(ctx context.Context, userID, tokenHash string, device *string, expiresAt time.Time) (string, error) {
	id := uuid.New().String()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dbo.refresh_tokens (id, user_id, token_hash, device_info, created_at, expires_at, is_revoked)
		VALUES (@p1,@p2,@p3,@p4,@p5,@p6,0)
	`, id, userID, tokenHash, device, time.Now().UTC(), expiresAt)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetRefreshTokenRow finds a refresh token by hash
func (r *UserRepo) GetRefreshTokenRow(ctx context.Context, tokenHash string) (*RefreshTokenRow, error) {
	row := r.db.QueryRowContext(ctx, `
	    SELECT id, user_id, token_hash, device_info, created_at, expires_at, is_revoked
	    FROM dbo.refresh_tokens WHERE token_hash = @p1
	`, tokenHash)
	var rr RefreshTokenRow
	var device sql.NullString
	if err := row.Scan(&rr.ID, &rr.UserID, &rr.TokenHash, &device, &rr.CreatedAt, &rr.ExpiresAt, &rr.Revoked); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if device.Valid {
		rr.Device = &device.String
	}
	return &rr, nil
}

func (r *UserRepo) RevokeRefreshToken(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE dbo.refresh_tokens SET is_revoked = 1 WHERE id = @p1`, id)
	return err
}

// RotateRefreshToken revokes old and inserts new token in a transaction
func (r *UserRepo) RotateRefreshToken(ctx context.Context, oldID, newTokenHash string, newExpiry time.Time) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	_, err = tx.ExecContext(ctx, `UPDATE dbo.refresh_tokens SET is_revoked = 1 WHERE id = @p1`, oldID)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	newID := uuid.New().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO dbo.refresh_tokens (id, user_id, token_hash, created_at, expires_at, is_revoked)
		SELECT @p1, user_id, @p2, SYSUTCDATETIME(), @p3, 0 FROM dbo.refresh_tokens WHERE id = @p4
	`, newID, newTokenHash, newExpiry, oldID)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return newID, nil
}

// helper for optional sql.NullString building from *string
func sqlNullString(p *string) interface{} {
	if p == nil {
		return nil
	}
	return *p
}
