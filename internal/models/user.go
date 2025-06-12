package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FullName string             `bson:"fullName" json:"fullName"`
	Email    string             `bson:"email" json:"email"`
	Password string             `bson:"password" json:"-"`  // Hide from JSON responses
	Role     string             `bson:"role" json:"role"`   // "client", "assistant", "dentist"
	Phone    string             `bson:"phone" json:"phone"` // Optional, can be empty
}
