package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Todo struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Title       string             `json:"title" bson:"title"`
	Completed   bool               `json:"completed" bson:"completed"`
	Description string             `json:"description" bson:"description"`
	Date        string             `json:"date" bson:"date"`
}
