package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Username    string             `json:"username" bson:"username"`
	Email       string             `json:"email" bson:"email"`
	PhoneNumber string             `json:"phone_number" bson:"phone_number"`
	ClerkId     string             `json:"clerk_id" bson:"clerk_id"`
	UpdatedAt   time.Time          `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty"`
}

func (u *User) GetID() *primitive.ObjectID {
	if u == nil {
		return nil
	}
	if u.ID == primitive.NilObjectID {
		return nil
	}
	return &u.ID
}

type ClerkUserEvent struct {
	Data   ClerkUser `json:"data"`
	Object string    `json:"object"`
	Type   string    `json:"type"`
}

type ClerkUser struct {
	Birthday       string `json:"birthday"`
	CreatedAt      int64  `json:"created_at"`
	EmailAddresses []struct {
		EmailAddress string        `json:"email_address"`
		Id           string        `json:"id"`
		LinkedTo     []interface{} `json:"linked_to"`
		Object       string        `json:"object"`
		Verification struct {
			Status   string `json:"status"`
			Strategy string `json:"strategy"`
		} `json:"verification"`
	} `json:"email_addresses"`
	ExternalAccounts      []interface{} `json:"external_accounts"`
	ExternalId            string        `json:"external_id"`
	FirstName             string        `json:"first_name"`
	Gender                string        `json:"gender"`
	Id                    string        `json:"id"`
	LastName              string        `json:"last_name"`
	LastSignInAt          int64         `json:"last_sign_in_at"`
	Object                string        `json:"object"`
	PasswordEnabled       bool          `json:"password_enabled"`
	PhoneNumbers          []interface{} `json:"phone_numbers"`
	PrimaryEmailAddressId string        `json:"primary_email_address_id"`
	PrimaryPhoneNumberId  interface{}   `json:"primary_phone_number_id"`
	PrimaryWeb3WalletId   interface{}   `json:"primary_web3_wallet_id"`
	PrivateMetadata       struct{}      `json:"private_metadata"`
	ProfileImageUrl       string        `json:"profile_image_url"`
	PublicMetadata        struct{}      `json:"public_metadata"`
	TwoFactorEnabled      bool          `json:"two_factor_enabled"`
	UnsafeMetadata        struct{}      `json:"unsafe_metadata"`
	UpdatedAt             int64         `json:"updated_at"`
	Username              interface{}   `json:"username"`
	Web3Wallets           []interface{} `json:"web3_wallets"`
}

func (c ClerkUser) GetEmail() string {
	if c.EmailAddresses[0].EmailAddress != "" {
		return c.EmailAddresses[0].EmailAddress
	}
	return ""
}

func (c ClerkUser) GetPhoneNumber() string {
	if c.PhoneNumbers[0] != nil {
		return c.PhoneNumbers[0].(string)
	}
	return ""
}

func (c ClerkUser) NewDBUser() User {
	return User{
		ID:          primitive.NewObjectID(),
		Username:    c.Username.(string),
		Email:       c.GetEmail(),
		PhoneNumber: c.GetPhoneNumber(),
		ClerkId:     c.Id,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

type ClerkUserDeleted struct {
	Data struct {
		Deleted bool   `json:"deleted"`
		Id      string `json:"id"`
		Object  string `json:"object"`
	} `json:"data"`
	Object string `json:"object"`
	Type   string `json:"type"`
}
