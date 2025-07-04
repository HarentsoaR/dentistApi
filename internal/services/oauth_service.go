package services

import (
	"context"
	"errors"
	"fmt"

	"log"

	"time"

	"github.com/harentsoaR/dentist-api/internal/config"
	"github.com/harentsoaR/dentist-api/internal/models"
	"github.com/harentsoaR/dentist-api/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// OAuthService provides methods for handling OAuth users and tokens
type OAuthService struct {
	userCollection *mongo.Collection
}

// NewOAuthService returns a new instance of OAuthService
func NewOAuthService(db *mongo.Database) *OAuthService {
	return &OAuthService{
		userCollection: db.Collection("users"),
	}
}

// FindOrCreateUser finds a user by provider or email, or creates a new one if not found
func (s *OAuthService) FindOrCreateUser(provider string, userInfo *config.GoogleUser) (*models.User, error) {
	// Try to find user by provider ID first
	var user models.User
	err := s.userCollection.FindOne(
		context.Background(),
		bson.M{"provider": provider, "providerId": userInfo.ID},
	).Decode(&user)

	// If user not found, try to find by email
	if err != nil {
		err = s.userCollection.FindOne(
			context.Background(),
			bson.M{"email": userInfo.Email},
		).Decode(&user)

		// If still not found, create a new user
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				// Create a new user
				newUser := &models.User{
					ID:         primitive.NewObjectID(),
					Email:      userInfo.Email,
					FullName:   userInfo.Name,
					Avatar:     userInfo.Picture,
					Provider:   "google",
					ProviderID: userInfo.ID,
					Role:       "client",
				}

				_, err = s.userCollection.InsertOne(context.Background(), newUser)
				if err != nil {
					log.Printf("Error creating new user: %v", err)
					return nil, fmt.Errorf("failed to create user: %v", err)
				}

				log.Printf("Created new user via OAuth: %s (%s)", newUser.Email, newUser.ID.Hex())
				return newUser, nil
			}
			// Some other error occurred
			log.Printf("Error finding user by email %s: %v", userInfo.Email, err)
			return nil, fmt.Errorf("failed to find user: %v", err)
		}
	}

	// User exists, update their information if needed
	update := bson.M{
		"$set": bson.M{
			"fullName":   userInfo.Name,
			"avatar":     userInfo.Picture,
			"provider":   "google",
			"providerId": userInfo.ID,
			"updatedAt":  time.Now(),
		},
	}

	_, err = s.userCollection.UpdateOne(context.Background(), bson.M{"_id": user.ID}, update)
	if err != nil {
		log.Printf("Error updating user %s: %v", user.ID.Hex(), err)
		return nil, fmt.Errorf("failed to update user: %v", err)
	}

	// Refresh the user data
	err = s.userCollection.FindOne(context.Background(), bson.M{"_id": user.ID}).Decode(&user)
	if err != nil {
		log.Printf("Error refreshing user data %s: %v", user.ID.Hex(), err)
		return nil, fmt.Errorf("failed to refresh user data: %v", err)
	}

	log.Printf("Logged in existing user via OAuth: %s (%s)", user.Email, user.ID.Hex())
	return &user, nil
}

// GenerateJWT creates a JWT token for the given user
func (s *OAuthService) GenerateJWT(user *models.User) (string, error) {
	token, err := utils.GenerateJWT(user.ID.Hex(), user.Role)
	if err != nil {
		log.Printf("Error generating JWT: %v", err)
		return "", fmt.Errorf("error generating token")
	}
	return token, nil
}
