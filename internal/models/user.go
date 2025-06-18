package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FullName     string             `bson:"fullName" json:"fullName"`
	Email        string             `bson:"email" json:"email"`
	Password     string             `bson:"password,omitempty" json:"-"` // Hide from JSON responses, omitempty for OAuth users
	Role         string             `bson:"role" json:"role"` // "client", "assistant", "dentist"
	Phone        string             `bson:"phone" json:"phone"` // Optional, can be empty
	Provider     string             `bson:"provider" json:"provider"` // "email", "google", etc.
	ProviderID   string             `bson:"providerId,omitempty" json:"providerId,omitempty"` // ID from the provider
	Avatar       string             `bson:"avatar,omitempty" json:"avatar,omitempty"`
	RefreshToken string             `bson:"refreshToken,omitempty" json:"-"`
}
