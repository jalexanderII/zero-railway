package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TransactionDetails is a DB Serialization of Proto TransactionDetails
type TransactionDetails struct {
	Address         string `json:"address" bson:"address"`
	City            string `json:"city" bson:"city"`
	State           string `json:"state" bson:"state"`
	Zipcode         string `json:"zipcode" bson:"zipcode"`
	Country         string `json:"country" bson:"country"`
	StoreNumber     string `json:"store_number" bson:"store_number"`
	ReferenceNumber string `json:"reference_number" bson:"reference_number"`
}

// Transaction is a DB Serialization of Proto Transaction
type Transaction struct {
	ID                   primitive.ObjectID  `json:"id,omitempty" bson:"_id,omitempty"`
	PlaidTransactionId   string              `json:"plaid_transaction_id" bson:"plaid_transaction_id"`
	AccountId            primitive.ObjectID  `json:"account_id" bson:"account_id"`
	PlaidAccountId       string              `json:"plaid_account_id" bson:"plaid_account_id"`
	UserId               primitive.ObjectID  `json:"user_id" bson:"user_id"`
	TransactionType      string              `json:"transaction_type" bson:"transaction_type"`
	PendingTransactionId string              `json:"pending_transaction_id" bson:"pending_transaction_id"`
	CategoryId           string              `json:"category_id" bson:"category_id"`
	Category             []string            `json:"category" bson:"category"`
	TransactionDetails   *TransactionDetails `json:"transaction_details" bson:"transaction_details"`
	Name                 string              `json:"name" bson:"name"`
	OriginalDescription  string              `json:"original_description" bson:"original_description"`
	Amount               float64             `json:"amount" bson:"amount"`
	IsoCurrencyCode      string              `json:"iso_currency_code" bson:"iso_currency_code"`
	Date                 string              `json:"date" bson:"date"`
	Pending              bool                `json:"pending" bson:"pending"`
	MerchantName         string              `json:"merchant_name" bson:"merchant_name"`
	PaymentChannel       string              `json:"payment_channel" bson:"payment_channel"`
	AuthorizedDate       string              `json:"authorized_date" bson:"authorized_date"`
	PrimaryCategory      string              `json:"primary_category" bson:"primary_category"`
	DetailedCategory     string              `json:"detailed_category" bson:"detailed_category"`
	UpdatedAt            time.Time           `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
	CreatedAt            time.Time           `json:"created_at,omitempty" bson:"created_at,omitempty"`
	InPlan               bool                `json:"in_plan" bson:"in_plan"`
}
