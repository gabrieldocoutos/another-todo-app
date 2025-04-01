package main

import (
	"bestodo/db"
	"bestodo/handlers"
	"bestodo/middleware"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

var database *mongo.Database

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize MongoDB connection
	database = db.InitDB()

	// Create Gin router
	r := gin.Default()

	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Auth routes
	auth := r.Group("/api/auth")
	{
		auth.POST("/signup", handlers.HandleSignUp(database))
		auth.POST("/signin", handlers.HandleSignIn(database))
	}

	// Todo routes (protected)
	todoHandler := handlers.NewTodoHandler(database)
	todos := r.Group("/api/todos")
	todos.Use(middleware.AuthMiddleware())
	{
		todos.GET("", todoHandler.GetTodos)
		todos.POST("", todoHandler.CreateTodo)
		todos.PATCH("/:id", todoHandler.ToggleTodoStatus)
		todos.DELETE("/:id", todoHandler.DeleteTodo)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
} 