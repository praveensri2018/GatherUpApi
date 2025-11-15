/* Place: backend/go/repository/user_repo.go */
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"gatherup/models"

	"github.com/google/uuid"
)

// UserRepo manages user-related DB operations.
// It logs info messages to infoLogger and errors to errorLogger (both also print to console).
type UserRepo struct {
	db          *sql.DB
	infoLogger  *log.Logger
	errorLogger *log.Logger
}

// RefreshTokenRow is a lightweight struct representing refresh_tokens row.
type RefreshTokenRow struct {
	ID        string
	UserID    string
	TokenHash string
	Device    *string
	CreatedAt time.Time
	ExpiresAt time.Time
	Revoked   bool
}

// NewUserRepo constructs a UserRepo. You may provide nil loggers to use defaults
// (default: info -> stdout, error -> logs/error.log + stdout).
// Place: call this from bootstrap (backend/go/cmd/server/main.go) instead of the old constructor.
func NewUserRepo(db *sql.DB, infoLogger, errorLogger *log.Logger) *UserRepo {
	if infoLogger == nil || errorLogger == nil {
		dInfo, dErr := defaultLoggers()
		if infoLogger == nil {
			infoLogger = dInfo
		}
		if errorLogger == nil {
			errorLogger = dErr
		}
	}
	return &UserRepo{db: db, infoLogger: infoLogger, errorLogger: errorLogger}
}

// defaultLoggers returns (infoLogger, errorLogger).
// errorLogger writes to a persistent file "logs/error.log" (create folder if needed) and also to stdout.
func defaultLoggers() (*log.Logger, *log.Logger) {
	// info -> stdout
	info := log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lmsgprefix)

	// ensure log directory exists
	if err := os.MkdirAll("logs", 0o755); err != nil {
		// fallback: print to stdout and return same logger for error
		info.Printf("failed to create logs dir: %v", err)
		errLogger := log.New(io.MultiWriter(os.Stdout), "ERROR: ", log.LstdFlags|log.Lmsgprefix)
		return info, errLogger
	}

	// open error log file for append
	f, err := os.OpenFile("logs/error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		// fallback to stdout if file can't be opened
		info.Printf("failed to open error log file: %v", err)
		errLogger := log.New(io.MultiWriter(os.Stdout), "ERROR: ", log.LstdFlags|log.Lmsgprefix)
		return info, errLogger
	}

	// error logger writes to file AND stdout for convenience
	errOut := io.MultiWriter(f, os.Stdout)
	errLogger := log.New(errOut, "ERROR: ", log.LstdFlags|log.Lmsgprefix)

	return info, errLogger
}

/* --------------------- Repository methods --------------------- */

// CreateUser inserts a new user (profile only) and returns generated id.
func (r *UserRepo) CreateUser(ctx context.Context, u *models.User) (string, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	r.infoLogger.Printf("CreateUser: starting id=%s mobile=%v", id, u.MobileNumber)

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO dbo.users (id, mobile_number, mobile_normalized, email, username, created_at, is_deleted)
        VALUES (@p1, @p2, @p3, @p4, @p5, @p6, 0)
    `, id, sqlNullString(u.MobileNumber), sqlNullString(u.MobileNormalized), sqlNullString(u.Email), sqlNullString(u.Username), now)
	if err != nil {
		r.errorLogger.Printf("CreateUser: exec failed id=%s err=%v", id, err)
		return "", fmt.Errorf("create user failed: %w", err)
	}
	r.infoLogger.Printf("CreateUser: inserted id=%s", id)
	return id, nil
}

// CreateUserWithPassword - create user row AND credential row in a single transaction.
func (r *UserRepo) CreateUserWithPassword(ctx context.Context, mobileRaw, mobileNorm, passwordHash string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.errorLogger.Printf("CreateUserWithPassword: begin tx failed: %v", err)
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userID := uuid.New().String()
	now := time.Now().UTC()

	r.infoLogger.Printf("CreateUserWithPassword: creating userID=%s mobile=%s", userID, mobileRaw)

	// Insert into users (profile only)
	if _, err = tx.ExecContext(ctx, `
        INSERT INTO dbo.users (id, mobile_number, mobile_normalized, created_at, is_deleted)
        VALUES (@p1, @p2, @p3, @p4, 0)
    `, userID, mobileRaw, mobileNorm, now); err != nil {
		r.errorLogger.Printf("CreateUserWithPassword: insert users failed userID=%s err=%v", userID, err)
		return "", err
	}

	// Insert into user_credentials
	if _, err = tx.ExecContext(ctx, `
        INSERT INTO dbo.user_credentials (user_id, credential_type, credential_identifier, password_hash, created_at, is_deleted)
        VALUES (@p1, @p2, @p3, @p4, @p5, 0)
    `, userID, "password", mobileNorm, passwordHash, now); err != nil {
		r.errorLogger.Printf("CreateUserWithPassword: insert credentials failed userID=%s err=%v", userID, err)
		return "", err
	}

	if err := tx.Commit(); err != nil {
		r.errorLogger.Printf("CreateUserWithPassword: commit failed userID=%s err=%v", userID, err)
		return "", err
	}
	r.infoLogger.Printf("CreateUserWithPassword: success userID=%s", userID)
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
			r.infoLogger.Printf("GetByEmailOrMobile: not found identifier=%s", identifier)
			return nil, nil
		}
		r.errorLogger.Printf("GetByEmailOrMobile: scan failed identifier=%s err=%v", identifier, err)
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
	r.infoLogger.Printf("GetByEmailOrMobile: found id=%s identifier=%s", u.ID, identifier)
	return u, nil
}

// GetByID returns profile by user id (no password)
func (r *UserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	// validate id shape (fast-fail)
	if _, err := uuid.Parse(id); err != nil {
		r.errorLogger.Printf("GetByID: invalid id format=%q err=%v", id, err)
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// request canonical string form for id to avoid driver raw-bytes
	row := r.db.QueryRowContext(ctx, `
        SELECT CONVERT(nvarchar(36), id) as id,
               mobile_number, mobile_normalized, email, username, created_at, updated_at, is_deleted
        FROM dbo.users WHERE id = @p1 AND is_deleted = 0
    `, id)

	u := &models.User{}
	var idStr sql.NullString
	var mobile, mobileNorm, email, username sql.NullString
	var updatedAt sql.NullTime

	if err := row.Scan(&idStr, &mobile, &mobileNorm, &email, &username, &u.CreatedAt, &updatedAt, &u.IsDeleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.infoLogger.Printf("GetByID: not found id=%s", id)
			return nil, nil
		}
		r.errorLogger.Printf("GetByID: scan failed id=%s err=%v", id, err)
		return nil, err
	}

	if !idStr.Valid {
		r.errorLogger.Printf("GetByID: missing id (unexpected) for id=%s", id)
		return nil, fmt.Errorf("user row missing id")
	}
	// assign canonical string id
	u.ID = idStr.String
	// validate (defensive)
	if _, err := uuid.Parse(u.ID); err != nil {
		r.errorLogger.Printf("GetByID: invalid id returned u.ID=%q err=%v", u.ID, err)
		return nil, fmt.Errorf("invalid user id in row: %w", err)
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

	r.infoLogger.Printf("GetByID: found id=%s", u.ID)
	return u, nil
}

// GetCredentialByIdentifier returns credential row for credential_type='password' and the linked user id.
// Returns (userID, passwordHash, nil) if found, ("","",nil) if not found.
func (r *UserRepo) GetCredentialByIdentifier(ctx context.Context, identifier string) (string, string, error) {
	var uid sql.NullString
	var ph sql.NullString

	row := r.db.QueryRowContext(ctx, `
        SELECT CONVERT(nvarchar(36), user_id), password_hash
        FROM dbo.user_credentials
        WHERE credential_type = 'password' AND credential_identifier = @p1 AND is_deleted = 0
    `, identifier)

	if err := row.Scan(&uid, &ph); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.infoLogger.Printf("GetCredentialByIdentifier: not found identifier=%s", identifier)
			return "", "", nil
		}
		r.errorLogger.Printf("GetCredentialByIdentifier: scan failed identifier=%s err=%v", identifier, err)
		return "", "", err
	}

	userID := ""
	if uid.Valid {
		userID = uid.String
		if _, err := uuid.Parse(userID); err != nil {
			r.errorLogger.Printf("GetCredentialByIdentifier: stored user_id not uuid identifier=%s user_id=%q err=%v", identifier, userID, err)
			return "", "", fmt.Errorf("stored user id invalid: %w", err)
		}
	}

	pwHash := ""
	if ph.Valid {
		pwHash = ph.String
	}
	r.infoLogger.Printf("GetCredentialByIdentifier: found user_id=%s identifier=%s", userID, identifier)
	return userID, pwHash, nil
}

// SaveRefreshToken stores refresh token hash. Validates userID before DB write.
func (r *UserRepo) SaveRefreshToken(ctx context.Context, userID, tokenHash string, device *string, expiresAt time.Time) (string, error) {
	// validate userID is a UUID before DB write
	if _, err := uuid.Parse(userID); err != nil {
		r.errorLogger.Printf("SaveRefreshToken: invalid user id format: %v (userID=%q len=%d)", err, userID, len(userID))
		return "", fmt.Errorf("invalid user id format: %w", err)
	}

	id := uuid.New().String()
	now := time.Now().UTC()

	// Log shape only (do NOT log real token contents)
	r.infoLogger.Printf("SaveRefreshToken: creating id=%s userID=%s devicePresent=%v expiresAt=%v",
		id, userID, device != nil && *device != "", expiresAt)

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO dbo.refresh_tokens (id, user_id, token_hash, device_info, created_at, expires_at, is_revoked)
        VALUES (@p1, @p2, @p3, @p4, @p5, @p6, 0)
    `, id, userID, tokenHash, device, now, expiresAt)
	if err != nil {
		r.errorLogger.Printf("SaveRefreshToken: exec failed id=%s userID=%s err=%v", id, userID, err)
		return "", fmt.Errorf("save refresh token failed: %w", err)
	}

	r.infoLogger.Printf("SaveRefreshToken: saved id=%s userID=%s", id, userID)
	return id, nil
}

// GetRefreshTokenRow finds a refresh token by hash
func (r *UserRepo) GetRefreshTokenRow(ctx context.Context, tokenHash string) (*RefreshTokenRow, error) {
	// ask SQL Server to return canonical strings for both id and user_id
	row := r.db.QueryRowContext(ctx, `
        SELECT CONVERT(nvarchar(36), id) as id,
               CONVERT(nvarchar(36), user_id) as user_id,
               token_hash, device_info, created_at, expires_at, is_revoked
        FROM dbo.refresh_tokens WHERE token_hash = @p1
    `, tokenHash)

	var rr RefreshTokenRow
	var device sql.NullString
	var uid sql.NullString
	var idStr sql.NullString

	if err := row.Scan(&idStr, &uid, &rr.TokenHash, &device, &rr.CreatedAt, &rr.ExpiresAt, &rr.Revoked); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.infoLogger.Printf("GetRefreshTokenRow: not found tokenHashLen=%d", len(tokenHash))
			return nil, nil
		}
		r.errorLogger.Printf("GetRefreshTokenRow: scan failed tokenHashLen=%d err=%v", len(tokenHash), err)
		return nil, err
	}

	if !idStr.Valid {
		r.errorLogger.Printf("GetRefreshTokenRow: row has empty id for tokenHash=%s", tokenHash)
		return nil, fmt.Errorf("refresh token row missing id")
	}
	rr.ID = idStr.String

	if device.Valid {
		rr.Device = &device.String
	}

	if uid.Valid {
		rr.UserID = uid.String
		// defensive validation
		if _, err := uuid.Parse(rr.UserID); err != nil {
			r.errorLogger.Printf("GetRefreshTokenRow: invalid user id in row rr.ID=%s userID=%q err=%v", rr.ID, rr.UserID, err)
			return nil, fmt.Errorf("invalid user id in refresh token row: %w", err)
		}
	} else {
		r.errorLogger.Printf("GetRefreshTokenRow: row has empty user_id rr.ID=%s", rr.ID)
		return nil, fmt.Errorf("refresh token row missing user id")
	}

	// validate returned id is a UUID
	if _, err := uuid.Parse(rr.ID); err != nil {
		r.errorLogger.Printf("GetRefreshTokenRow: invalid id in row rr.ID=%q err=%v", rr.ID, err)
		return nil, fmt.Errorf("invalid refresh token id: %w", err)
	}

	r.infoLogger.Printf("GetRefreshTokenRow: found id=%s userID=%s", rr.ID, rr.UserID)
	return &rr, nil
}

// RevokeRefreshToken marks a refresh token revoked
func (r *UserRepo) RevokeRefreshToken(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		r.errorLogger.Printf("RevokeRefreshToken: invalid id format=%q err=%v", id, err)
		return fmt.Errorf("invalid id: %w", err)
	}
	_, err := r.db.ExecContext(ctx, `UPDATE dbo.refresh_tokens SET is_revoked = 1 WHERE id = @p1`, id)
	if err != nil {
		r.errorLogger.Printf("RevokeRefreshToken: exec failed id=%s err=%v", id, err)
		return err
	}
	r.infoLogger.Printf("RevokeRefreshToken: revoked id=%s", id)
	return nil
}

// RotateRefreshToken revokes old and inserts a new token in a transaction
func (r *UserRepo) RotateRefreshToken(ctx context.Context, oldID, newTokenHash string, newExpiry time.Time) (string, error) {
	// validate IDs
	if _, err := uuid.Parse(oldID); err != nil {
		r.errorLogger.Printf("RotateRefreshToken: invalid oldID=%q err=%v", oldID, err)
		return "", fmt.Errorf("invalid old id: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.errorLogger.Printf("RotateRefreshToken: begin tx failed oldID=%s err=%v", oldID, err)
		return "", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `UPDATE dbo.refresh_tokens SET is_revoked = 1 WHERE id = @p1`, oldID); err != nil {
		r.errorLogger.Printf("RotateRefreshToken: revoke old failed oldID=%s err=%v", oldID, err)
		return "", err
	}
	newID := uuid.New().String()
	// use CONVERT to ensure user_id is inserted as canonical nvarchar(36)
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO dbo.refresh_tokens (id, user_id, token_hash, created_at, expires_at, is_revoked)
        SELECT @p1, CONVERT(nvarchar(36), user_id), @p2, SYSUTCDATETIME(), @p3, 0 FROM dbo.refresh_tokens WHERE id = @p4
    `, newID, newTokenHash, newExpiry, oldID); err != nil {
		r.errorLogger.Printf("RotateRefreshToken: insert new failed oldID=%s newID=%s err=%v", oldID, newID, err)
		return "", err
	}
	if err := tx.Commit(); err != nil {
		r.errorLogger.Printf("RotateRefreshToken: commit failed oldID=%s newID=%s err=%v", oldID, newID, err)
		return "", err
	}
	committed = true
	r.infoLogger.Printf("RotateRefreshToken: rotated oldID=%s -> newID=%s", oldID, newID)
	return newID, nil
}

/* helper for optional sql.NullString building from *string */
func sqlNullString(p *string) interface{} {
	if p == nil {
		return nil
	}
	return *p
}
