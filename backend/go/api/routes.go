package api

import (
	"net/http"

	"gatherup/auth"
	"gatherup/repository"
	"gatherup/service"
	"gatherup/storage"

	"github.com/go-chi/chi/v5"
)

func WireRouter(
	userRepo *repository.UserRepo,
	socialRepo *repository.SocialRepo,
	r2 *storage.R2Client,
	jwtMgr *auth.JWTManager,
	authSvc *service.AuthService,
) http.Handler {

	r := chi.NewRouter()

	/* ---------------- AUTH VERIFY ---------------- */

	verifyFn := func(token string) (string, error) {
		claims, err := jwtMgr.Verify(token)
		if err != nil {
			return "", err
		}
		return claims.UserID, nil
	}

	/* ---------------- HANDLERS ---------------- */

	// Auth / User (UserRepo)
	authHandler := NewAuthHandler(authSvc)
	userHandler := NewUserHandler(userRepo)

	// Phase-2 (SocialRepo only)
	postHandler := NewPostHandler(socialRepo, r2)
	commentHandler := NewCommentHandler(socialRepo)
	feedHandler := NewFeedHandler(socialRepo)
	socialHandler := NewSocialHandler(socialRepo)

	/* ---------------- AUTH ROUTES ---------------- */

	r.Post("/auth/register",
		RateLimitAuth(0.2, 5)(http.HandlerFunc(authHandler.Register)).ServeHTTP)

	r.Post("/auth/login",
		RateLimitAuth(0.2, 5)(http.HandlerFunc(authHandler.Login)).ServeHTTP)

	r.Post("/auth/refresh",
		RateLimitAuth(0.5, 10)(http.HandlerFunc(authHandler.Refresh)).ServeHTTP)

	r.Post("/auth/revoke",
		RateLimitAuth(0.5, 10)(http.HandlerFunc(authHandler.Revoke)).ServeHTTP)

	r.Post("/auth/forgot/send-otp",
		RateLimitAuth(0.1, 5)(http.HandlerFunc(authHandler.SendForgotOTP)).ServeHTTP)

	r.Post("/auth/forgot/verify-otp",
		RateLimitAuth(0.2, 10)(http.HandlerFunc(authHandler.VerifyForgotOTP)).ServeHTTP)

	r.Post("/auth/forgot/reset",
		RateLimitAuth(0.2, 10)(http.HandlerFunc(authHandler.ResetPassword)).ServeHTTP)

	/* ---------------- PROTECTED ROUTES ---------------- */

	r.Group(func(r chi.Router) {
		r.Use(WithAuth(verifyFn))

		// USER
		r.Get("/api/me", userHandler.Me)

		// FEED
		r.Get("/posts/feed/personalized", feedHandler.PersonalizedFeed)

		// POSTS
		r.Post("/posts", postHandler.Create)
		r.Get("/posts", postHandler.List)
		r.Get("/posts/{id}", postHandler.Get)
		r.Patch("/posts/{id}", postHandler.Update)
		r.Delete("/posts/{id}", postHandler.Delete)

		// POST MEDIA (placeholder)
		r.Post("/posts/{id}/media", postHandler.UploadMedia)

		// POST REACTIONS
		r.Post("/posts/{id}/reactions", postHandler.React)
		r.Delete("/posts/{id}/reactions", postHandler.Unreact)

		// COMMENTS
		r.Get("/posts/{id}/comments", commentHandler.List)
		r.Post("/posts/{id}/comments", commentHandler.Create)
		r.Patch("/comments/{id}", commentHandler.Update)
		r.Delete("/comments/{id}", commentHandler.Delete)

		// COMMENT REACTIONS
		r.Post("/comments/{id}/reactions", commentHandler.React)
		r.Delete("/comments/{id}/reactions", commentHandler.Unreact)

		// BOOKMARK
		r.Post("/posts/{id}/bookmark", socialHandler.Bookmark)
		r.Delete("/posts/{id}/bookmark", socialHandler.Unbookmark)

		// FOLLOW / UNFOLLOW
		r.Post("/users/{id}/follow", socialHandler.Follow)
		r.Post("/users/{id}/unfollow", socialHandler.Unfollow)

		// BLOCK / UNBLOCK
		r.Post("/blocks", socialHandler.Block)
		r.Delete("/blocks", socialHandler.Unblock)

		// REPORT
		r.Post("/posts/{id}/report", postHandler.Report)
		r.Post("/comments/{id}/report", commentHandler.Report)

		r.Get("/bookmarks", socialHandler.ListBookmarks)

		r.Get("/users/{id}/followers", socialHandler.ListFollowers)
		r.Get("/users/{id}/following", socialHandler.ListFollowing)

		r.Get("/blocks", socialHandler.ListBlockedUsers)

		r.Get("/users/me/posts", postHandler.ListMyPosts)

	})

	/* ---------------- HEALTH ---------------- */

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return r
}
