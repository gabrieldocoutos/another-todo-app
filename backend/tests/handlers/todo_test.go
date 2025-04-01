package handlers_test

import (
	"bestodo/handlers"
	"bestodo/models"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	testDB     *mongo.Database
	testClient *mongo.Client
)

func setupTestDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to test database
	var err error
	testClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	assert.NoError(t, err)

	err = testClient.Ping(ctx, nil)
	assert.NoError(t, err)

	testDB = testClient.Database("bestodo_test")
}

func teardownTestDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Drop the test database
	err := testDB.Drop(ctx)
	assert.NoError(t, err)

	// Close the connection
	err = testClient.Disconnect(ctx)
	assert.NoError(t, err)
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestCreateTodo(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	router := setupRouter()
	handler := handlers.NewTodoHandler(testDB)

	// Create a test user ID
	userID := primitive.NewObjectID()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.Hex())
		c.Next()
	})

	router.POST("/todos", handler.CreateTodo)

	tests := []struct {
		name       string
		input      map[string]interface{}
		wantStatus int
	}{
		{
			name:       "valid todo",
			input:      map[string]interface{}{"title": "Test Todo"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty title",
			input:      map[string]interface{}{"title": ""},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest("POST", "/todos", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusCreated {
				var response models.Todo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.input["title"], response.Title)
				assert.Equal(t, userID, response.UserID)
				assert.False(t, response.IsCompleted)
			}
		})
	}
}

func TestGetTodos(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	router := setupRouter()
	handler := handlers.NewTodoHandler(testDB)

	// Create a test user ID
	userID := primitive.NewObjectID()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.Hex())
		c.Next()
	})

	router.GET("/todos", handler.GetTodos)

	// Insert some test todos
	todos := []models.Todo{
		{
			ID:          primitive.NewObjectID(),
			UserID:      userID,
			Title:       "Todo 1",
			IsCompleted: false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          primitive.NewObjectID(),
			UserID:      userID,
			Title:       "Todo 2",
			IsCompleted: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, todo := range todos {
		_, err := testDB.Collection("todos").InsertOne(ctx, todo)
		assert.NoError(t, err)
	}

	req := httptest.NewRequest("GET", "/todos", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.Todo
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)
}

func TestToggleTodoStatus(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	router := setupRouter()
	handler := handlers.NewTodoHandler(testDB)

	// Create a test user ID
	userID := primitive.NewObjectID()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.Hex())
		c.Next()
	})

	router.PATCH("/todos/:id", handler.ToggleTodoStatus)

	// Create a test todo
	todo := models.Todo{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Title:       "Test Todo",
		IsCompleted: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := testDB.Collection("todos").InsertOne(ctx, todo)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		todoID     string
		input      map[string]interface{}
		wantStatus int
	}{
		{
			name:       "toggle to completed",
			todoID:     todo.ID.Hex(),
			input:      map[string]interface{}{"isCompleted": true},
			wantStatus: http.StatusOK,
		},
		{
			name:       "toggle to incomplete",
			todoID:     todo.ID.Hex(),
			input:      map[string]interface{}{"isCompleted": false},
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent todo",
			todoID:     primitive.NewObjectID().Hex(),
			input:      map[string]interface{}{"isCompleted": true},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest("PATCH", "/todos/"+tt.todoID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var response models.Todo
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.input["isCompleted"], response.IsCompleted)
			}
		})
	}
}

func TestDeleteTodo(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	router := setupRouter()
	handler := handlers.NewTodoHandler(testDB)

	// Create a test user ID
	userID := primitive.NewObjectID()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.Hex())
		c.Next()
	})

	router.DELETE("/todos/:id", handler.DeleteTodo)

	// Create a test todo
	todo := models.Todo{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Title:       "Test Todo",
		IsCompleted: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := testDB.Collection("todos").InsertOne(ctx, todo)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		todoID     string
		wantStatus int
	}{
		{
			name:       "delete existing todo",
			todoID:     todo.ID.Hex(),
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete non-existent todo",
			todoID:     primitive.NewObjectID().Hex(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/todos/"+tt.todoID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Todo deleted successfully", response["message"])

				// Verify todo was actually deleted
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				var deletedTodo models.Todo
				err = testDB.Collection("todos").FindOne(ctx, bson.M{"_id": todo.ID}).Decode(&deletedTodo)
				assert.Equal(t, mongo.ErrNoDocuments, err)
			}
		})
	}
} 