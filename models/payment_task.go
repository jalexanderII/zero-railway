package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PaymentTask is a DB Serialization of Proto PaymentTask
type PaymentTask struct {
	ID           primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserId       primitive.ObjectID `json:"user_id" bson:"user_id"`
	AccountId    primitive.ObjectID `json:"account_id" bson:"account_id"`
	Amount       float64            `json:"amount" bson:"amount"`
	Transactions []string           `json:"transactions" bson:"transactions,omitempty"`
	UpdatedAt    time.Time          `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
	CreatedAt    time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty"`
}
