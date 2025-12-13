package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

/*
SocialRepo
----------
Handles:
- Posts
- Feed
- Comments
- Reactions
- Follow / Unfollow
- Bookmark
- Block
- Report
- LIST APIs (followers, following, bookmarks, blocks)
*/

type SocialRepo struct {
	db *sql.DB
}

func NewSocialRepo(db *sql.DB) *SocialRepo {
	return &SocialRepo{db: db}
}

/* ============================================================
   POSTS
============================================================ */

func placeholders(n int) string {
	p := make([]string, n)
	for i := range p {
		p[i] = "@p" + strconv.Itoa(i+1)
	}
	return strings.Join(p, ",")
}

func (r *SocialRepo) getMediaByPostID(
	ctx context.Context,
	postID string,
) ([]map[string]any, error) {

	rows, err := r.db.QueryContext(ctx, `
		SELECT media_url, media_type, file_size, sort_order
		FROM dbo.post_media
		WHERE post_id = @p1 AND is_deleted = 0
		ORDER BY sort_order ASC
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []map[string]any
	for rows.Next() {
		var url, mtype string
		var size int64
		var order int

		rows.Scan(&url, &mtype, &size, &order)

		list = append(list, map[string]any{
			"media_url":  url,
			"media_type": mtype,
			"file_size":  size,
			"sort_order": order,
		})
	}
	return list, nil
}

func (r *SocialRepo) getMediaByPostIDs(
	ctx context.Context,
	postIDs []string,
) (map[string][]map[string]any, error) {

	if len(postIDs) == 0 {
		return map[string][]map[string]any{}, nil
	}

	args := make([]any, len(postIDs))
	for i, id := range postIDs {
		args[i] = id
	}

	query := `
		SELECT post_id, media_url, media_type, file_size, sort_order
		FROM dbo.post_media
		WHERE post_id IN (` + placeholders(len(postIDs)) + `)
		  AND is_deleted = 0
		ORDER BY sort_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]map[string]any)

	for rows.Next() {
		var postID, url, mtype string
		var size int64
		var order int

		rows.Scan(&postID, &url, &mtype, &size, &order)

		result[postID] = append(result[postID], map[string]any{
			"media_url":  url,
			"media_type": mtype,
			"file_size":  size,
			"sort_order": order,
		})
	}

	return result, nil
}

func (r *SocialRepo) CreatePost(ctx context.Context, userID, title, body, kind string) (string, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return "", err
	}

	postID := uuid.New().String()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dbo.posts
			(id, author_id, title, body, kind, created_at, is_deleted)
		VALUES
			(@p1, @p2, @p3, @p4, @p5, SYSUTCDATETIME(), 0)
	`, postID, userID, title, body, kind)

	return postID, err
}

func (r *SocialRepo) ListPosts(ctx context.Context) ([]map[string]any, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT CONVERT(nvarchar(36), id),
		       CONVERT(nvarchar(36), author_id),
		       title, body, kind, created_at
		FROM dbo.posts
		WHERE is_deleted = 0
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []map[string]any
	var postIDs []string

	for rows.Next() {
		var id, authorID, title, body, kind string
		var createdAt any

		rows.Scan(&id, &authorID, &title, &body, &kind, &createdAt)

		postIDs = append(postIDs, id)

		posts = append(posts, map[string]any{
			"id":         id,
			"author_id":  authorID,
			"title":      title,
			"body":       body,
			"kind":       kind,
			"created_at": createdAt,
			"media":      []any{},
		})
	}

	mediaMap, err := r.getMediaByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, err
	}

	for _, p := range posts {
		p["media"] = mediaMap[p["id"].(string)]
	}

	return posts, nil
}

func (r *SocialRepo) GetPostByID(
	ctx context.Context,
	postID string,
) (map[string]any, error) {

	row := r.db.QueryRowContext(ctx, `
		SELECT CONVERT(nvarchar(36), id),
		       CONVERT(nvarchar(36), author_id),
		       title, body, kind, created_at
		FROM dbo.posts
		WHERE id = @p1 AND is_deleted = 0
	`, postID)

	var id, authorID, title, body, kind string
	var createdAt any

	if err := row.Scan(&id, &authorID, &title, &body, &kind, &createdAt); err != nil {
		return nil, err
	}

	media, err := r.getMediaByPostID(ctx, id)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":         id,
		"author_id":  authorID,
		"title":      title,
		"body":       body,
		"kind":       kind,
		"created_at": createdAt,
		"media":      media,
	}, nil
}

func (r *SocialRepo) UpdatePost(ctx context.Context, postID, title, body string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE dbo.posts
		SET title = @p2,
		    body = @p3,
		    updated_at = SYSUTCDATETIME()
		WHERE id = @p1 AND is_deleted = 0
	`, postID, title, body)

	return err
}

func (r *SocialRepo) DeletePost(ctx context.Context, postID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE dbo.posts
		SET is_deleted = 1, deleted_at = SYSUTCDATETIME()
		WHERE id = @p1
	`, postID)

	return err
}

/* ============================================================
   FEED
============================================================ */

func (r *SocialRepo) GetPersonalizedFeed(ctx context.Context, userID string) ([]map[string]any, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, err
	}
	return r.ListPosts(ctx)
}

/* ============================================================
   COMMENTS
============================================================ */

func (r *SocialRepo) ListComments(ctx context.Context, postID string) ([]map[string]any, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT CONVERT(nvarchar(36), id),
		       CONVERT(nvarchar(36), author_id),
		       body, created_at
		FROM dbo.comments
		WHERE post_id = @p1 AND is_deleted = 0
		ORDER BY created_at ASC
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []map[string]any
	for rows.Next() {
		var id, authorID, body string
		var createdAt any

		rows.Scan(&id, &authorID, &body, &createdAt)

		list = append(list, map[string]any{
			"id":         id,
			"author_id":  authorID,
			"body":       body,
			"created_at": createdAt,
		})
	}
	return list, nil
}

func (r *SocialRepo) CreateComment(ctx context.Context, postID, userID, body string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dbo.comments
			(post_id, author_id, body, created_at, is_deleted)
		VALUES
			(@p1, @p2, @p3, SYSUTCDATETIME(), 0)
	`, postID, userID, body)

	return err
}

func (r *SocialRepo) UpdateComment(ctx context.Context, commentID, body string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE dbo.comments
		SET body = @p2,
		    updated_at = SYSUTCDATETIME()
		WHERE id = @p1 AND is_deleted = 0
	`, commentID, body)

	return err
}

func (r *SocialRepo) DeleteComment(ctx context.Context, commentID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE dbo.comments
		SET is_deleted = 1, deleted_at = SYSUTCDATETIME()
		WHERE id = @p1
	`, commentID)

	return err
}

/* ============================================================
   REACTIONS
============================================================ */

func (r *SocialRepo) ReactToPost(ctx context.Context, postID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dbo.post_reactions
			(post_id, user_id, reaction_type_id, created_at, is_deleted)
		VALUES
			(@p1, @p2, 1, SYSUTCDATETIME(), 0)
	`, postID, userID)

	return err
}

func (r *SocialRepo) UnreactPost(ctx context.Context, postID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE dbo.post_reactions
		SET is_deleted = 1, deleted_at = SYSUTCDATETIME()
		WHERE post_id = @p1 AND user_id = @p2 AND is_deleted = 0
	`, postID, userID)

	return err
}

func (r *SocialRepo) ReactToComment(ctx context.Context, commentID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dbo.comment_reactions
			(comment_id, user_id, reaction_type_id, created_at, is_deleted)
		VALUES
			(@p1, @p2, 1, SYSUTCDATETIME(), 0)
	`, commentID, userID)

	return err
}

func (r *SocialRepo) UnreactComment(ctx context.Context, commentID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE dbo.comment_reactions
		SET is_deleted = 1, deleted_at = SYSUTCDATETIME()
		WHERE comment_id = @p1 AND user_id = @p2 AND is_deleted = 0
	`, commentID, userID)

	return err
}

/* ============================================================
   SOCIAL LIST APIs (NEW)
============================================================ */

func (r *SocialRepo) ListBookmarkedPosts(ctx context.Context, userID string) ([]map[string]any, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT CONVERT(nvarchar(36), p.id),
		       CONVERT(nvarchar(36), p.author_id),
		       p.title, p.body, p.kind, p.created_at
		FROM dbo.post_bookmarks b
		JOIN dbo.posts p ON p.id = b.post_id
		WHERE b.user_id = @p1 AND b.is_deleted = 0
		ORDER BY p.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []map[string]any
	for rows.Next() {
		var id, authorID, title, body, kind string
		var createdAt any
		rows.Scan(&id, &authorID, &title, &body, &kind, &createdAt)

		list = append(list, map[string]any{
			"id":         id,
			"author_id":  authorID,
			"title":      title,
			"body":       body,
			"kind":       kind,
			"created_at": createdAt,
		})
	}
	return list, nil
}

func (r *SocialRepo) ListFollowers(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT CONVERT(nvarchar(36), follower_id)
		FROM dbo.user_follows
		WHERE following_id = @p1 AND is_deleted = 0
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		list = append(list, id)
	}
	return list, nil
}

func (r *SocialRepo) ListFollowing(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT CONVERT(nvarchar(36), following_id)
		FROM dbo.user_follows
		WHERE follower_id = @p1 AND is_deleted = 0
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		list = append(list, id)
	}
	return list, nil
}

func (r *SocialRepo) ListBlockedUsers(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT CONVERT(nvarchar(36), blocked_user_id)
		FROM dbo.blocks
		WHERE user_id = @p1 AND is_deleted = 0
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		list = append(list, id)
	}
	return list, nil
}

/* ============================================================
   REPORT
============================================================ */

func (r *SocialRepo) Report(ctx context.Context, reporterID, targetType, targetID, reason string) error {
	if _, err := uuid.Parse(reporterID); err != nil {
		return err
	}
	if targetType == "" {
		return errors.New("target type required")
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dbo.reports
			(reporter_id, target_type, target_id, reason, status, created_at)
		VALUES
			(@p1, @p2, @p3, @p4, 0, SYSUTCDATETIME())
	`, reporterID, targetType, targetID, reason)

	return err
}

func (r *SocialRepo) FollowUser(ctx context.Context, userID, targetID string) error {
	if _, err := uuid.Parse(userID); err != nil {
		return err
	}
	if _, err := uuid.Parse(targetID); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx, `
		IF EXISTS (
			SELECT 1 FROM dbo.user_follows
			WHERE follower_id = @p1 AND following_id = @p2
		)
		BEGIN
			UPDATE dbo.user_follows
			SET is_deleted = 0,
			    deleted_at = NULL,
			    created_at = SYSUTCDATETIME()
			WHERE follower_id = @p1 AND following_id = @p2
		END
		ELSE
		BEGIN
			INSERT INTO dbo.user_follows
				(follower_id, following_id, created_at, is_deleted)
			VALUES
				(@p1, @p2, SYSUTCDATETIME(), 0)
		END
	`, userID, targetID)

	return err
}

func (r *SocialRepo) UnfollowUser(
	ctx context.Context,
	userID string,
	targetID string,
) error {

	_, err := r.db.ExecContext(ctx, `
		UPDATE dbo.user_follows
		SET is_deleted = 1,
		    deleted_at = SYSUTCDATETIME()
		WHERE follower_id = @p1
		  AND following_id = @p2
		  AND is_deleted = 0
	`, userID, targetID)

	return err
}

func (r *SocialRepo) BookmarkPost(ctx context.Context, postID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		IF EXISTS (
			SELECT 1 FROM dbo.post_bookmarks
			WHERE post_id = @p1 AND user_id = @p2
		)
		BEGIN
			UPDATE dbo.post_bookmarks
			SET is_deleted = 0,
			    deleted_at = NULL,
			    created_at = SYSUTCDATETIME()
			WHERE post_id = @p1 AND user_id = @p2
		END
		ELSE
		BEGIN
			INSERT INTO dbo.post_bookmarks
				(post_id, user_id, created_at, is_deleted)
			VALUES
				(@p1, @p2, SYSUTCDATETIME(), 0)
		END
	`, postID, userID)

	return err
}

func (r *SocialRepo) UnbookmarkPost(
	ctx context.Context,
	postID string,
	userID string,
) error {

	_, err := r.db.ExecContext(ctx, `
		UPDATE dbo.post_bookmarks
		SET is_deleted = 1,
		    deleted_at = SYSUTCDATETIME()
		WHERE post_id = @p1
		  AND user_id = @p2
		  AND is_deleted = 0
	`, postID, userID)

	return err
}

func (r *SocialRepo) BlockUser(
	ctx context.Context,
	userID string,
	blockedUserID string,
	reason *string,
) error {

	if _, err := uuid.Parse(userID); err != nil {
		return err
	}
	if _, err := uuid.Parse(blockedUserID); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx, `
		IF EXISTS (
			SELECT 1 FROM dbo.blocks
			WHERE user_id = @p1 AND blocked_user_id = @p2
		)
		BEGIN
			UPDATE dbo.blocks
			SET is_deleted = 0,
			    deleted_at = NULL,
			    reason = @p3,
			    created_at = SYSUTCDATETIME()
			WHERE user_id = @p1 AND blocked_user_id = @p2
		END
		ELSE
		BEGIN
			INSERT INTO dbo.blocks
				(user_id, blocked_user_id, reason, created_at, is_deleted)
			VALUES
				(@p1, @p2, @p3, SYSUTCDATETIME(), 0)
		END
	`, userID, blockedUserID, reason)

	return err
}

func (r *SocialRepo) UnblockUser(
	ctx context.Context,
	userID string,
	blockedUserID string,
) error {

	_, err := r.db.ExecContext(ctx, `
		UPDATE dbo.blocks
		SET is_deleted = 1,
		    deleted_at = SYSUTCDATETIME()
		WHERE user_id = @p1
		  AND blocked_user_id = @p2
		  AND is_deleted = 0
	`, userID, blockedUserID)

	return err
}

func (r *SocialRepo) ListPostsByUser(
	ctx context.Context,
	userID string,
) ([]map[string]any, error) {

	if _, err := uuid.Parse(userID); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT CONVERT(nvarchar(36), id),
		       CONVERT(nvarchar(36), author_id),
		       title, body, kind, created_at
		FROM dbo.posts
		WHERE author_id = @p1
		  AND is_deleted = 0
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []map[string]any
	for rows.Next() {
		var id, authorID, title, body, kind string
		var createdAt any

		rows.Scan(&id, &authorID, &title, &body, &kind, &createdAt)

		list = append(list, map[string]any{
			"id":         id,
			"author_id":  authorID,
			"title":      title,
			"body":       body,
			"kind":       kind,
			"created_at": createdAt,
		})
	}

	return list, nil
}

func (r *SocialRepo) AddPostMedia(
	ctx context.Context,
	postID string,
	mediaURL string,
	mediaType string,
	fileSize int64,
	sortOrder int,
) error {

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dbo.post_media
			(post_id,
			 media_url,
			 media_type,
			 file_size,
			 sort_order,
			 created_at,
			 is_deleted)
		VALUES
			(@p1, @p2, @p3, @p4, @p5, SYSUTCDATETIME(), 0)
	`,
		postID,
		mediaURL,
		mediaType,
		fileSize,
		sortOrder,
	)

	return err
}
