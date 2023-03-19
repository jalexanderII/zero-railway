package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	client "github.com/jalexanderII/zero-railway/app/clients"

	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SendSMSResponse struct {
	Successful   bool   `json:"successful"`
	ErrorMessage string `json:"error_message"`
}

// @Summary Notify all users of upcoming payment actions.
// @Description Check all users payment actions and notify for any approaching payments.
// @Tags planning
// @Accept */*
// @Produce json
// @Success 200 {object} []models.SendSMSResponse
// @Router /notify [get]
func NotifyUsersUpcomingPaymentActions(tc *client.TwilioClient, h *Handler, planningUrl string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		err := cleanUpStalePaymentPlans(h, planningUrl)
		if err != nil {
			h.L.Error("[Planning] error cleaning up old payment plans ", err.Error())
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error cleaning up old payment plans", err.Error())
		}

		url := fmt.Sprintf("%s/paymentactions", planningUrl)
		paymentActionsRequest := &models.GetAllUpcomingPaymentActionsRequest{
			Date: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
		upcomingPaymentActionsAllUsers, err := planningGetAllUpcomingPaymentActions(h, url, paymentActionsRequest)
		if err != nil {
			h.L.Error("error listing upcoming PaymentActions", err.Error())
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error listing upcoming PaymentActions", err.Error())
		}
		h.L.Info("upcomingPaymentActionsAllUsers", upcomingPaymentActionsAllUsers)
		userIds := upcomingPaymentActionsAllUsers.UserIds
		paymentActions := upcomingPaymentActionsAllUsers.PaymentActions

		// create map of UserID -> AccID -> Liability
		userAccLiabilities := make(map[string]map[string]float64)
		for idx := range paymentActions {
			_, created := userAccLiabilities[userIds[idx]]
			if !created {
				userAccLiabilities[userIds[idx]] = make(map[string]float64)
			}
			userAccLiabilities[userIds[idx]][paymentActions[idx].AccountId] += paymentActions[idx].Amount
		}

		// creates map of how to inform users
		userNotify := make(map[string]string)
		for userId, accLiab := range userAccLiabilities {
			totalLiab := 0.0
			for _, liab := range accLiab {
				totalLiab += liab
			}

			id, _ := primitive.ObjectIDFromHex(userId)
			userAccs, err := GetUserAccounts(h, &id)
			if err != nil {
				return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error listing accounts", err.Error())
			}
			totalDebit := GetDebitAccountBalance(h, &id)
			if err != nil {
				return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error getting debit balance", err.Error())
			}

			if totalDebit.CurrentBalance < totalLiab {
				userNotify[userId] = fmt.Sprintf("You are missing $%v for tomorrows upcoming total payment of $%v", totalLiab-totalDebit.CurrentBalance, totalLiab)
			} else {
				userNotify[userId] = fmt.Sprintf("You are all setup for tomorrows total payment of $%v", totalLiab)
			}
			for accId, liab := range accLiab {
				accName := ""
				for _, acc := range userAccs {
					if acc.ID.Hex() == accId {
						accName = acc.OfficialName
						break
					}
				}
				startingStr := fmt.Sprintf("For account %v: $%v \n", accName, liab)
				startingStr += userNotify[userId]
				userNotify[userId] = startingStr
			}
		}

		resps := make([]models.SendSMSResponse, 0)
		// send notifications to the appropriate user
		for userId, message := range userNotify {
			user, _ := h.GetUserByID(userId)
			resp, err := tc.SendSMS(user.PhoneNumber, message)
			if err != nil {
				h.L.Error("error sending SMS", err.Error())
				return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error sending SMS", resp)
			}
			resps = append(resps, *resp)
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "successfully notified users", resps)
	}
}

func planningGetAllUpcomingPaymentActions(h *Handler, url string, req *models.GetAllUpcomingPaymentActionsRequest) (*models.GetAllUpcomingPaymentActionsResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := h.H.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var result models.GetAllUpcomingPaymentActionsResponse

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func cleanUpStalePaymentPlans(h *Handler, planningUrl string) error {
	url := fmt.Sprintf("%s/cleanup", planningUrl)
	_, err := h.H.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	return nil
}
