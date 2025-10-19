package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Register/Login DTOs
type RegisterReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name"`
}
type LoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func registerHandler(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var exists User
	if err := DB.Where("email = ?", req.Email).First(&exists).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already used"})
		return
	}
	ph, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	u := User{
		UID:       uuid.NewString(),
		Email:     req.Email,
		Password:  string(ph),
		Name:      req.Name,
		CreatedAt: time.Now(),
	}
	if err := DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db create failed"})
		return
	}
	tokenStr, err := createJWT(u.UID, u.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token create failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": tokenStr, "uid": u.UID})
}

func loginHandler(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user User
	if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	tokenStr, err := createJWT(user.UID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token create failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenStr, "uid": user.UID})
}

func createJWT(uid, email string) (string, error) {
	secret := []byte(JwtSecret())
	claims := jwt.MapClaims{
		"uid":   uid,
		"email": email,
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}

// Event endpoints
type EventCreateReq struct {
	GameName        string `json:"gameName" binding:"required"`
	DateIso         string `json:"dateIso" binding:"required"`
	Location        string `json:"location"`
	IsOnline        bool   `json:"isOnline"`
	Description     string `json:"description"`
	MaxParticipants int    `json:"maxParticipants"`
}

func createEventHandler(c *gin.Context) {
	uid, _ := c.Get("uid")
	var req EventCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	parsed, err := time.Parse(time.RFC3339, req.DateIso)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dateIso, RFC3339 required"})
		return
	}
	ev := Event{
		UUID:            uuid.NewString(),
		HostUID:         uid.(string),
		GameName:        req.GameName,
		DateIso:         parsed,
		Location:        req.Location,
		IsOnline:        req.IsOnline,
		Description:     req.Description,
		MaxParticipants: req.MaxParticipants,
		Status:          "upcoming",
		CreatedAt:       time.Now(),
	}
	if err := DB.Create(&ev).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create event failed"})
		return
	}
	c.JSON(http.StatusCreated, ev)
}

func listEventsHandler(c *gin.Context) {
	var events []Event
	DB.Where("status = ?", "upcoming").Order("date_iso asc").Limit(100).Find(&events)
	c.JSON(http.StatusOK, events)
}

func getEventHandler(c *gin.Context) {
	u := c.Param("uuid")
	var ev Event
	if err := DB.Where("uuid = ?", u).First(&ev).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, ev)
}

func joinEventHandler(c *gin.Context) {
	uid, _ := c.Get("uid")
	eventUUID := c.Param("uuid")
	var body struct {
		Message string `json:"message"`
	}
	_ = c.ShouldBindJSON(&body)
	jr := JoinRequest{
		UUID:      uuid.NewString(),
		EventUUID: eventUUID,
		UserUID:   uid.(string),
		Message:   body.Message,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	if err := DB.Create(&jr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "join create failed"})
		return
	}
	c.JSON(http.StatusCreated, jr)
}
