package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnnualPercentageRates struct {
	AprPercentage        float64 `json:"apr_percentage" bson:"apr_percentage"`
	AprType              string  `json:"apr_type" bson:"apr_type"`
	BalanceSubjectToApr  float64 `json:"balance_subject_to_apr" bson:"balance_subject_to_apr"`
	InterestChargeAmount float64 `json:"interest_charge_amount" bson:"interest_charge_amount"`
}

type Account struct {
	ID                     primitive.ObjectID       `json:"id,omitempty" bson:"_id,omitempty"`
	PlaidAccountId         string                   `json:"plaid_account_id" bson:"plaid_account_id"`
	UserId                 primitive.ObjectID       `json:"user_id" bson:"user_id"`
	Name                   string                   `json:"name" bson:"name"`
	OfficialName           string                   `json:"official_name" bson:"official_name"`
	Type                   string                   `json:"type" bson:"type"`
	Subtype                string                   `json:"subtype" bson:"subtype"`
	AvailableBalance       float64                  `json:"available_balance" bson:"available_balance"`
	CurrentBalance         float64                  `json:"current_balance" bson:"current_balance"`
	CreditLimit            float64                  `json:"credit_limit" bson:"credit_limit"`
	IsoCurrencyCode        string                   `json:"iso_currency_code" bson:"iso_currency_code"`
	AnnualPercentageRate   []*AnnualPercentageRates `json:"annual_percentage_rate" bson:"annual_percentage_rate"`
	IsOverdue              bool                     `json:"is_overdue" bson:"is_overdue"`
	LastPaymentAmount      float64                  `json:"last_payment_amount" bson:"last_payment_amount"`
	LastStatementIssueDate string                   `json:"last_statement_issue_date" bson:"last_statement_issue_date"`
	LastStatementBalance   float64                  `json:"last_statement_balance" bson:"last_statement_balance"`
	MinimumPaymentAmount   float64                  `json:"minimum_payment_amount" bson:"minimum_payment_amount"`
	NextPaymentDueDate     string                   `json:"next_payment_due_date" bson:"next_payment_due_date"`
	UpdatedAt              time.Time                `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
	CreatedAt              time.Time                `json:"created_at,omitempty" bson:"created_at,omitempty"`
}

type GetDebitAccountBalanceResponse struct {
	AvailableBalance float64 `json:"available_balance"`
	CurrentBalance   float64 `json:"current_balance"`
}
