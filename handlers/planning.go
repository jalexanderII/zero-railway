package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"net/http"
)

type ListPaymentPlanResponse struct {
	PaymentPlans []models.PaymentPlanS `json:"payment_plans"`
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

		accounts, err := GetUserAccounts(h, user.GetID())
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user accounts not found", err)
		}

		totalCredit := 0.0
		for _, account := range accounts {
			totalCredit += account.CurrentBalance
		}

		var totalDebit = 0.0
		debitAccBalance := GetDebitAccountBalance(h, user.GetID())
		if debitAccBalance != nil {
			totalDebit = debitAccBalance.CurrentBalance
		}

		var totalPlanAmount = 0.0
		url := fmt.Sprintf("%s/payment_plans/%s", planningUrl, user.GetID().Hex())
		plans, err := planningGetUserPaymentPlans(url)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "user payment plans not found", err)
		}
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

type WaterfallMonth struct {
	AccountToAmounts map[string]float64 `json:"account_to_amounts"`
}

type WaterfallOverviewResponse struct {
	MonthlyWaterfall []WaterfallMonth `json:"monthly_waterfall"`
}

type Series struct {
	Name string    `json:"name"`
	Data []float32 `json:"data"`
}

// @Summary Get user waterfall data.
// @Description Create a waterfall from users payment plans.
// @Tags planning
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} Series
// @Router /waterfall/:email [get]
func GetWaterfall(h *Handler, planningUrl string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err)
		}
		url := fmt.Sprintf("%s/waterfall/%s", planningUrl, user.GetID().Hex())

		overview, err := planningGetWaterfall(url)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "Error fetching user's waterfall", err)
		}

		accountSeries := make(map[string]Series)
		for idx, waterfallMonth := range overview.MonthlyWaterfall {
			for name, value := range waterfallMonth.AccountToAmounts {
				if series, ok := accountSeries[name]; ok {
					series.Data[idx] = float32(value)
				} else {
					accountSeries[name] = Series{Name: name, Data: make([]float32, 12)}
					accountSeries[name].Data[idx] = float32(value)
				}
			}
		}
		var response []Series
		for _, series := range accountSeries {
			response = append(response, series)
		}

		return FiberJsonResponse(c, fiber.StatusOK, "success", "waterfall", response)
	}
}

func planningGetWaterfall(url string) (*WaterfallOverviewResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	var result WaterfallOverviewResponse

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
