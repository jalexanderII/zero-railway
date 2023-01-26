package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"net/http"
)

type ListPaymentPlanResponse struct {
	PaymentPlans []models.PaymentPlan `json:"payment_plans"`
}

type KPI struct {
	Debit        float64 `json:"debit"`
	Credit       float64 `json:"credit"`
	PaymentPlans float64 `json:"payment_plans"`
}

// @Summary Get a user KPIs.
// @Description fetch a KPIs for a single user.
// @Tags planning
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} KPI
// @Router /kpi/:email [get]
func GetKPIs(h *Handler, planningUrl string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err)
		}
		h.L.Info("user found", "user", user)

		accounts, err := GetUserAccounts(h, user.GetID())
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user accounts not found", err)
		}
		h.L.Info("user accounts found", "accounts", accounts)

		totalCredit := 0.0
		for _, account := range accounts {
			totalCredit += account.CurrentBalance
		}
		h.L.Info("total credit", "total_credit", totalCredit)

		var totalDebit = 0.0
		debitAccBalance := GetDebitAccountBalance(h, user.GetID())
		if debitAccBalance != nil {
			totalDebit = debitAccBalance.CurrentBalance
		}
		h.L.Info("total debit", "total_debit", totalDebit)

		var totalPlanAmount = 0.0
		url := fmt.Sprintf("%s/payment_plans/%s", planningUrl, user.GetID().Hex())
		plans, err := planningGetUserPaymentPlans(url)
		if err != nil {
			h.L.Error("error getting user payment plans", "error", err)
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "user payment plans not found", err)
		}
		h.L.Info("user payment plans found", "plans", plans)
		for _, plan := range plans.PaymentPlans {
			totalPlanAmount += plan.Amount
		}

		return FiberJsonResponse(
			c, fiber.StatusOK, "success", "account",
			KPI{Debit: totalDebit, Credit: totalCredit, PaymentPlans: totalPlanAmount},
		)
	}
}

func planningGetUserPaymentPlans(url string) (*ListPaymentPlanResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	var result ListPaymentPlanResponse

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

//func GetWaterfallOverview(client core.CoreClient, ctx context.Context) func(c *fiber.Ctx) error {
//	return func(c *fiber.Ctx) error {
//		user, err := GetUserByEmail(client, ctx, c)
//		if err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on fetching user's from email", "data": err})
//		}
//
//		overview, err := client.GetWaterfallOverview(ctx, &planning.GetUserOverviewRequest{UserId: user.GetId()})
//		if err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error fetching user's waterfall", "data": err})
//		}
//		type Series struct {
//			Name string    `json:"name"`
//			Data []float32 `json:"data"`
//		}
//
//		accountSeries := make(map[string]Series)
//		monthlyWaterfall := overview.GetMonthlyWaterfall()
//		for idx, WaterfallMonth := range monthlyWaterfall {
//			for name, value := range WaterfallMonth.GetAccountToAmounts() {
//				if series, ok := accountSeries[name]; ok {
//					series.Data[idx] = float32(value)
//				} else {
//					accountSeries[name] = Series{Name: name, Data: make([]float32, 12)}
//					accountSeries[name].Data[idx] = float32(value)
//				}
//			}
//		}
//		response := []Series{}
//		for _, series := range accountSeries {
//			response = append(response, series)
//		}
//
//		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": response})
//	}
//}
