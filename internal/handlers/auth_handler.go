// internal/handlers/auth_handler.go
package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/harentsoaR/dentist-api/internal/models"
	"github.com/harentsoaR/dentist-api/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RegisterUserRequest struct {
	FullName string `json:"fullName" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role"`
	Phone    string `json:"phone" binding:"required"` // Ajout du champ phone avec validation
}

// RegisterUser is a METHOD of the Handler struct.
func (h *Handler) RegisterUser(c *gin.Context) {
	var req RegisterUserRequest // Utilise notre nouvelle structure de requête

	// ShouldBindJSON va maintenant lire le mot de passe ET valider les champs
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Nous construisons maintenant manuellement notre modèle `User` pour la base de données
	log.Println("RegisterUser: Attempting to hash password...")
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	log.Println("RegisterUser: Password hashed successfully.")

	// Définir le rôle par défaut si non fourni
	role := req.Role
	if role == "" {
		role = "client"
	}

	user := models.User{
		ID:       primitive.NewObjectID(),
		FullName: req.FullName,
		Email:    req.Email,
		Password: hashedPassword, // Le mot de passe haché est maintenant stocké
		Role:     role,
		Phone:    req.Phone, // Ajout du champ phone
	}

	collection := h.DB.Collection("users")
	log.Println("RegisterUser: Attempting to insert user into database...")
	_, err = collection.InsertOne(context.TODO(), user)
	if err != nil {
		// Gérer le cas où l'email existe déjà
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "An account with this email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	log.Println("RegisterUser: User inserted successfully.")

	// Le tag `json:"-"` sur `user.Password` dans la structure `models.User`
	// empêchera le mot de passe haché d'être renvoyé ici. C'est parfait.
	c.JSON(http.StatusCreated, user)
}

// Login is a METHOD of the Handler struct.
func (h *Handler) Login(c *gin.Context) {
	var loginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var user models.User
	collection := h.DB.Collection("users")
	log.Println("Login: Attempting to find user by email...")
	err := collection.FindOne(context.TODO(), bson.M{"email": loginReq.Email}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	log.Println("Login: User found successfully.")

	log.Println("Login: Attempting to check password hash...")
	if !utils.CheckPasswordHash(loginReq.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	log.Println("Login: Password hash checked successfully.")

	log.Println("Login: Attempting to generate JWT...")
	token, err := utils.GenerateJWT(user.ID.Hex(), user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}
	log.Println("Login: JWT generated successfully.")

	// Don't send password back
	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// GetCurrentUser retrieves the profile of the currently authenticated user.
func (h *Handler) GetCurrentUser(c *gin.Context) {
	// Get userID from the context (set by auth middleware)
	userIDHex, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDHex.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	var user models.User
	collection := h.DB.Collection("users")
	err = collection.FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateCurrentUser allows a user to update their own profile (e.g., full name).
func (h *Handler) UpdateCurrentUser(c *gin.Context) {
	userIDHex, _ := c.Get("userID")
	userID, _ := primitive.ObjectIDFromHex(userIDHex.(string))

	// Define a struct for the update request to control what can be changed
	var req struct {
		FullName string `json:"fullName"`
		// Add other updatable fields here, e.g., Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Create the update document for MongoDB
	update := bson.M{
		"$set": bson.M{},
	}

	// Only add fields to the update if they were provided in the request
	if req.FullName != "" {
		update["$set"].(bson.M)["fullName"] = req.FullName
	}

	// If nothing to update, return
	if len(update["$set"].(bson.M)) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No update fields provided"})
		return
	}

	collection := h.DB.Collection("users")
	result, err := collection.UpdateOne(context.TODO(), bson.M{"_id": userID}, update)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}
