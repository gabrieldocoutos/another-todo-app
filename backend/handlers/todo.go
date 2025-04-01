package handlers

import (
	"bestodo/models"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TodoHandler struct {
	db *mongo.Database
}

func NewTodoHandler(db *mongo.Database) *TodoHandler {
	return &TodoHandler{db: db}
}

func (h *TodoHandler) GetTodos(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	cursor, err := h.db.Collection("todos").Find(context.Background(), bson.M{"userId": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching todos"})
		return
	}
	defer cursor.Close(context.Background())

	var todos []models.Todo
	if err = cursor.All(context.Background(), &todos); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding todos"})
		return
	}

	if todos == nil {
		todos = []models.Todo{} // Return empty array instead of null
	}

	c.JSON(http.StatusOK, todos)
}

func (h *TodoHandler) CreateTodo(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var input struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	todo := models.Todo{
		UserID:      userID,
		Title:       input.Title,
		IsCompleted: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result, err := h.db.Collection("todos").InsertOne(context.Background(), todo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating todo"})
		return
	}

	todo.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, todo)
}

func (h *TodoHandler) ToggleTodoStatus(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	todoID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})
		return
	}

	var input struct {
		IsCompleted bool `json:"isCompleted"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := bson.M{
		"_id":     todoID,
		"userId": userID,
	}

	update := bson.M{
		"$set": bson.M{
			"isCompleted": input.IsCompleted,
			"updatedAt":  time.Now(),
		},
	}

	var todo models.Todo
	err = h.db.Collection("todos").FindOneAndUpdate(
		context.Background(),
		filter,
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&todo)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating todo"})
		return
	}

	c.JSON(http.StatusOK, todo)
}

func (h *TodoHandler) DeleteTodo(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	todoID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})
		return
	}

	filter := bson.M{
		"_id":     todoID,
		"userId": userID,
	}

	result, err := h.db.Collection("todos").DeleteOne(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting todo"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Todo deleted successfully"})
} 