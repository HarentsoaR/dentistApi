package handlers

import (
	"github.com/harentsoaR/dentist-api/internal/services" // <-- Import the new service
	"go.mongodb.org/mongo-driver/mongo"
)

// STEP 1: Add NotificationSvc to the Handler struct.
// Now your "toolbox" has a slot for the notification service.
type Handler struct {
	DB              *mongo.Database
	NotificationSvc *services.NotificationService // <-- THIS IS THE NEW FIELD
}

// STEP 2: Update the NewHandler function to accept the new service.
// This is the "factory" that builds your handler.
func NewHandler(db *mongo.Database, notificationSvc *services.NotificationService) *Handler {
	// It now returns a Handler with BOTH the database and the notification service.
	return &Handler{
		DB:              db,
		NotificationSvc: notificationSvc, // <-- ASSIGN THE SERVICE HERE
	}
}

// NOTE: You should move your other handler functions (like RegisterUser, Login, etc.)
// into their own files in this package (e.g., `user_handler.go`) but make sure they are
// methods of this same `*Handler` struct.
// For example: func (h *Handler) RegisterUser(c *gin.Context) { ... }
