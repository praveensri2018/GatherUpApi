package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	InitDB()
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.POST("/register", registerHandler)
	r.POST("/login", loginHandler)
	r.GET("/events", listEventsHandler)
	r.GET("/events/:uuid", getEventHandler)

	auth := r.Group("/")
	auth.Use(JwtMiddleware())
	{
		auth.POST("/events", createEventHandler)
		auth.POST("/events/:uuid/join", joinEventHandler)
	}

	port := ServerPort()
	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
