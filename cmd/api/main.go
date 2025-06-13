package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/harentsoaR/dentist-api/internal/handlers"
	"github.com/harentsoaR/dentist-api/internal/middleware"
	"github.com/harentsoaR/dentist-api/internal/services" // Import the new service
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}
	log.Printf("MONGO_URI: %s", os.Getenv("MONGO_URI"))
	log.Printf("MONGO_DATABASE: %s", os.Getenv("MONGO_DATABASE"))
	log.Printf("API_PORT: %s", os.Getenv("API_PORT"))
	if os.Getenv("JWT_SECRET") != "" {
		log.Println("JWT_SECRET is SET.")
	} else {
		log.Println("JWT_SECRET is NOT SET.")
	}

	// --- Database Connection ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)
	db := client.Database(os.Getenv("MONGO_DATABASE"))
	log.Println("Successfully connected to MongoDB!")

	// --- Initialize Services ---
	notificationSvc := services.NewNotificationService()

	// --- Initialize Handlers with DB and Services ---
	h := handlers.NewHandler(db, notificationSvc)

	// --- Gin Router ---
	r := gin.Default()

	// ---  Middleware ---
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://dentaheal.netlify.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// --- Routes ---
	authRoutes := r.Group("/auth")
	{
		// Assuming you will move these handlers into the handlers package
		authRoutes.POST("/register", h.RegisterUser)
		authRoutes.POST("/login", h.Login)
	}

	apiRoutes := r.Group("/api")
	apiRoutes.Use(middleware.AuthMiddleware()) // Protect all /api routes
	{
		// Appointment Routes
		apiRoutes.GET("/appointments", h.GetAppointments)    // Get appointments with filters
		apiRoutes.POST("/appointments", h.CreateAppointment) // Create a new appointment
		apiRoutes.GET("/appointment/user/:id", h.GetAppointment)
		apiRoutes.PUT("/appointments/:id", h.UpdateAppointment)          // Update an appointment (dentist/staff)
		apiRoutes.PATCH("/appointments/:id/cancel", h.CancelAppointment) // Cancel an appointment (dentist/staff)

		// other existing routes
		apiRoutes.POST("/chat", h.HandleChat)
		apiRoutes.GET("/user/:id", h.GetCurrentUser)
		apiRoutes.PUT("/user/:id", h.UpdateCurrentUser)
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080" // Default port
	}
	log.Printf("Starting server on port %s", port)
	r.Run(":" + port)
}
