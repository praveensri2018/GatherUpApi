package api

import (
	"gatherup/auth"
	"gatherup/repository"
	"gatherup/service"
	"gatherup/storage"
	"net/http"

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

	verifyFn := func(token string) (string, error) {
		claims, err := jwtMgr.Verify(token)
		if err != nil {
			return "", err
		}
		return claims.UserID, nil
	}

	authHandler := NewAuthHandler(authSvc)
	userHandler := NewUserHandler(userRepo)
	postHandler := NewPostHandler(socialRepo, r2)
	commentHandler := NewCommentHandler(socialRepo)
	feedHandler := NewFeedHandler(socialRepo)
	socialHandler := NewSocialHandler(socialRepo)

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

	r.Group(func(r chi.Router) {
		r.Use(WithAuth(verifyFn))
		r.Get("/api/me", userHandler.Me)
		r.Get("/posts/feed/personalized", feedHandler.PersonalizedFeed)
		r.Post("/posts", postHandler.Create)
		r.Get("/posts", postHandler.List)
		r.Get("/posts/{id}", postHandler.Get)
		r.Patch("/posts/{id}", postHandler.Update)
		r.Delete("/posts/{id}", postHandler.Delete)
		r.Post("/posts/{id}/media", postHandler.UploadMedia)
		r.Post("/posts/{id}/reactions", postHandler.React)
		r.Delete("/posts/{id}/reactions", postHandler.Unreact)
		r.Get("/posts/{id}/comments", commentHandler.List)
		r.Post("/posts/{id}/comments", commentHandler.Create)
		r.Patch("/comments/{id}", commentHandler.Update)
		r.Delete("/comments/{id}", commentHandler.Delete)
		r.Post("/comments/{id}/reactions", commentHandler.React)
		r.Delete("/comments/{id}/reactions", commentHandler.Unreact)
		r.Post("/posts/{id}/bookmark", socialHandler.Bookmark)
		r.Delete("/posts/{id}/bookmark", socialHandler.Unbookmark)
		r.Post("/users/{id}/follow", socialHandler.Follow)
		r.Post("/users/{id}/unfollow", socialHandler.Unfollow)
		r.Post("/blocks", socialHandler.Block)
		r.Delete("/blocks", socialHandler.Unblock)
		r.Post("/posts/{id}/report", postHandler.Report)
		r.Post("/comments/{id}/report", commentHandler.Report)
		r.Get("/bookmarks", socialHandler.ListBookmarks)
		r.Get("/users/{id}/followers", socialHandler.ListFollowers)
		r.Get("/users/{id}/following", socialHandler.ListFollowing)
		r.Get("/blocks", socialHandler.ListBlockedUsers)
		r.Get("/users/me/posts", postHandler.ListMyPosts)
		r.Get("/users/{id}", userHandler.GetProfile)
		r.Patch("/users/me", userHandler.UpdateMe)
		r.Post("/users/me/avatar", userHandler.UpdateAvatar)
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return r
}
