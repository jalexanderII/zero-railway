package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"time"
)

// @Summary Create a Payment Plan for the user.
// @Description Create a payment plans for a specific user.
// @Tags paymentplan
// @Accept json
// @Param email path string true "User email" body models.GetPaymentPlanRequest true "Payment Plan request"
// @Produce json
// @Success 200 {object} []models.PaymentPlan
// @Router /paymentplan/:email [post]
func CreatePaymentPlan(h *Handler, planningUrl string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err)
		}

		currentDate := time.Now().Format("01.02.2006")
		var input models.GetPaymentPlanRequest
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Error on login request", "data": err})
		}
		input.UserId = user.GetID().Hex()

		accountInfoList := make([]models.AccountInfo, len(input.AccountInfo))
		for idx, accountInfo := range input.AccountInfo {
			accountInfoList[idx] = accountInfo
		}
		metaData := models.MetaData{
			PreferredPlanType:         input.MetaData.PreferredPlanType,
			PreferredTimelineInMonths: input.MetaData.PreferredTimelineInMonths,
			PreferredPaymentFreq:      input.MetaData.PreferredPaymentFreq,
		}
		paymentPlanResponse, err := GetPaymentPlan(h, &models.GetPaymentPlanRequest{AccountInfo: accountInfoList, UserId: input.UserId, MetaData: metaData, SavePlan: input.SavePlan}, planningUrl)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(err.Error())
		}

		responsePaymentPlans := make([]models.PaymentPlan, len(paymentPlanResponse.PaymentPlans))
		for idx, paymentPlan := range paymentPlanResponse.PaymentPlans {
			pp := CreateResponsePaymentPlan(paymentPlan)
			name := fmt.Sprintf("Plan_%v_%v_%v", idx+1, pp.UserId[len(pp.UserId)-4:], currentDate)
			pp.Name = name
			responsePaymentPlans[idx] = pp
		}

		return c.Status(fiber.StatusOK).JSON(responsePaymentPlans)
	}
}

func GetPaymentPlan(h *Handler, in *models.GetPaymentPlanRequest, planningUrl string) (*models.PaymentPlanResponse, error) {
	// create payment task from user inputs
	paymentTasks := make([]models.PaymentTask, len(in.AccountInfo))

	for idx, item := range in.AccountInfo {
		id, _ := primitive.ObjectIDFromHex(in.UserId)
		accId, _ := primitive.ObjectIDFromHex(item.AccountId)
		task := models.PaymentTask{
			UserId:    id,
			AccountId: accId,
			Amount:    item.Amount,
		}
		paymentTasks[idx] = task
	}

	// save payment tasks to DB
	listOfIds, err := CreateManyPaymentTask(h, paymentTasks)
	if err != nil {
		h.L.Error("[PaymentTask] Error creating PaymentTasks", "error", err)
		return nil, err
	}

	for idx, id := range listOfIds {
		pt, _ := GetPaymentTask(h, id)
		paymentTasks[idx] = *pt
	}

	// send payment tasks to planning to get payment plans
	url := fmt.Sprintf("%s/paymentplan", planningUrl)
	res, err := planningCreatePaymentPlan(url, &models.CreatePaymentPlanRequest{PaymentTasks: paymentTasks, MetaData: in.MetaData, SavePlan: in.SavePlan})
	if err != nil {
		return nil, err
	}
	return &models.PaymentPlanResponse{PaymentPlans: res.PaymentPlans}, nil
}

func planningCreatePaymentPlan(url string, req *models.CreatePaymentPlanRequest) (*models.PaymentPlanResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	var result models.PaymentPlanResponse

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func CreateManyPaymentTask(h *Handler, in []models.PaymentTask) ([]string, error) {
	// Map struct slice to interface slice as InsertMany accepts interface slice as parameter
	insertableList := make([]interface{}, len(in))
	for i, v := range in {
		v.ID = primitive.NewObjectID()
		insertableList[i] = v
	}

	// Perform InsertMany operation & validate against the error.
	insertManyResult, err := h.Db.InsertMany(h.C, insertableList)
	if err != nil {
		return nil, err
	}

	resp := make([]string, len(insertManyResult.InsertedIDs))
	for idx, id := range insertManyResult.InsertedIDs {
		ido := id.(primitive.ObjectID)
		resp[idx] = ido.Hex()
	}

	// Return success without any error.
	return resp, nil
}

func GetPaymentTask(h *Handler, ID string) (*models.PaymentTask, error) {
	var paymentTask models.PaymentTask
	id, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		return nil, err
	}

	filter := bson.D{{Key: "_id", Value: id}}
	err = h.Db.FindOne(h.C, filter).Decode(&paymentTask)
	if err != nil {
		return nil, err
	}
	return &paymentTask, nil
}

// CreateResponsePaymentPlan Takes in a model and returns a serializer
func CreateResponsePaymentPlan(paymentTaskModel *models.PaymentPlan) models.PaymentPlan {
	paymentActions := make([]models.PaymentAction, len(paymentTaskModel.PaymentAction))
	for idx, paymentAction := range paymentTaskModel.PaymentAction {
		paymentActions[idx] = models.PaymentAction{
			AccountId:       paymentAction.AccountId,
			Amount:          paymentAction.Amount,
			TransactionDate: paymentAction.TransactionDate,
			Status:          paymentAction.Status,
		}
	}
	return models.PaymentPlan{
		Name:             "",
		PaymentPlanId:    paymentTaskModel.PaymentPlanId,
		UserId:           paymentTaskModel.UserId,
		PaymentTaskId:    paymentTaskModel.PaymentTaskId,
		Timeline:         paymentTaskModel.Timeline,
		PaymentFreq:      paymentTaskModel.PaymentFreq,
		Amount:           paymentTaskModel.Amount,
		AmountPerPayment: paymentTaskModel.AmountPerPayment,
		PlanType:         paymentTaskModel.PlanType,
		EndDate:          paymentTaskModel.EndDate,
		Active:           paymentTaskModel.Active,
		Status:           paymentTaskModel.Status,
		PaymentAction:    paymentActions,
	}
}
