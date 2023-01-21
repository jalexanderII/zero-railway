package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Username    string             `json:"username" bson:"username"`
	Email       string             `json:"email" bson:"email"`
	PhoneNumber string             `json:"phone_number" bson:"phone_number"`
	UpdatedAt   time.Time          `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty"`
}
