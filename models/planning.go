package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type PlanType int32

const (
	PlanType_PLAN_TYPE_UNKNOWN            PlanType = 0
	PlanType_PLAN_TYPE_OPTIM_CREDIT_SCORE PlanType = 1
	PlanType_PLAN_TYPE_MIN_FEES           PlanType = 2
)

type PaymentFrequency int32

const (
	PaymentFrequency_PAYMENT_FREQUENCY_UNKNOWN   PaymentFrequency = 0
	PaymentFrequency_PAYMENT_FREQUENCY_WEEKLY    PaymentFrequency = 1
	PaymentFrequency_PAYMENT_FREQUENCY_BIWEEKLY  PaymentFrequency = 2
	PaymentFrequency_PAYMENT_FREQUENCY_MONTHLY   PaymentFrequency = 3
	PaymentFrequency_PAYMENT_FREQUENCY_QUARTERLY PaymentFrequency = 4
)

type PaymentStatus int32

const (
	PaymentStatus_PAYMENT_STATUS_UNKNOWN PaymentStatus = 0
	// Payment plan is in good standing and all payments are current
	PaymentStatus_PAYMENT_STATUS_CURRENT PaymentStatus = 1
	// This plan is fully paid
	PaymentStatus_PAYMENT_STATUS_COMPLETED PaymentStatus = 2
	// The user has requested this payment to be cancelled
	PaymentStatus_PAYMENT_STATUS_CANCELLED PaymentStatus = 3
	// The payment plan is not current because the user has missed a payment and still have not paid it
	PaymentStatus_PAYMENT_STATUS_IN_DEFAULT PaymentStatus = 4
)

type PaymentActionStatus int32

const (
	PaymentActionStatus_PAYMENT_ACTION_STATUS_UNKNOWN PaymentActionStatus = 0
	// Payment is pending
	PaymentActionStatus_PAYMENT_ACTION_STATUS_PENDING PaymentActionStatus = 1
	// payment is completed
	PaymentActionStatus_PAYMENT_ACTION_STATUS_COMPLETED PaymentActionStatus = 2
	// payment defaulted
	PaymentActionStatus_PAYMENT_ACTION_STATUS_IN_DEFAULT PaymentActionStatus = 3
)

type PaymentPlan struct {
	ID               primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	PaymentPlanId    string             `json:"payment_plan_id,omitempty"`
	UserId           string             `json:"user_id,omitempty"`
	PaymentTaskId    []string           `json:"payment_task_id,omitempty"`
	Amount           float64            `json:"amount,omitempty"`
	Timeline         float64            `json:"timeline,omitempty"`
	PaymentFreq      PaymentFrequency   `json:"payment_freq,omitempty"`
	AmountPerPayment float64            `json:"amount_per_payment,omitempty"`
	PlanType         PlanType           `json:"plan_type,omitempty"`
	EndDate          string             `json:"end_date,omitempty"`
	Active           bool               `json:"active,omitempty"`
	Status           string             `json:"status,omitempty"`
	PaymentAction    []PaymentAction    `json:"payment_action,omitempty"`
}

//Status           PaymentStatus             `json:"status,omitempty"`

type PaymentAction struct {
	ID              primitive.ObjectID  `json:"id,omitempty" bson:"_id,omitempty"`
	AccountId       string              `json:"account_id,omitempty"`
	Amount          float64             `json:"amount,omitempty"`
	TransactionDate string              `json:"transaction_date,omitempty"`
	Status          PaymentActionStatus `json:"status,omitempty"`
}
