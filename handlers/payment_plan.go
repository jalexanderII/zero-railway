package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jalexanderII/zero-railway/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// @Summary Create a Payment Plan for the user.
// @Description Create a payment plans for a specific user.
// @Tags paymentplan
// @Accept json
// @Param email path string true "User email"
// @Param payment_plan_request body models.GetPaymentPlanRequest true "Payment Plan request"
// @Produce json
// @Success 200 {object} []models.PaymentPlan
// @Router /paymentplan/:email [post]
func CreatePaymentPlan(h, transactionHandler *Handler, planningUrl string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")
		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found ", err.Error())
		}

		currentDate := time.Now().Format("01.02.2006")
		var input models.GetPaymentPlanRequest
		if err = c.BodyParser(&input); err != nil {
			return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "error parsing request ", err.Error())
		}
		input.UserId = user.GetID().Hex()

		// accountInfoList := make([]models.AccountInfo, len(input.AccountInfo))
		// for idx, accountInfo := range input.AccountInfo {
		// 	accountInfoList[idx] = accountInfo
		// }
		// copy(accountInfoList, input.AccountInfo)
		metaData := models.MetaData{
			PreferredPlanType:         input.MetaData.PreferredPlanType,
			PreferredTimelineInMonths: input.MetaData.PreferredTimelineInMonths,
			PreferredPaymentFreq:      input.MetaData.PreferredPaymentFreq,
		}
		paymentPlanResponse, err := GetPaymentPlan(h, &models.GetPaymentPlanRequest{AccountInfo: input.AccountInfo, UserId: input.UserId, MetaData: metaData, SavePlan: input.SavePlan}, planningUrl)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error getting payment plan ", err.Error())
		}

		if input.SavePlan {
			err = MarkTrxnAsPlanned(transactionHandler, input.AccountInfo)
			if err != nil {
				return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "error marking transactions as in plan ", err.Error())
			}
		}

		responsePaymentPlans := make([]models.PaymentPlan, len(paymentPlanResponse.PaymentPlans))
		for idx, paymentPlan := range paymentPlanResponse.PaymentPlans {
			pp := CreateResponsePaymentPlan(paymentPlan)
			name := fmt.Sprintf("Plan_%v_%v_%v", idx+1, pp.UserId[len(pp.UserId)-4:], currentDate)
			pp.Name = name
			responsePaymentPlans[idx] = pp
		}

		return FiberJsonResponse(c, fiber.StatusOK, "success", "payment plan created", responsePaymentPlans)
	}
}

func MarkTrxnAsPlanned(h *Handler, info []models.AccountInfo) error {
	for _, accountInfo := range info {
		for _, trxnId := range accountInfo.TransactionIds {
			oid, err := primitive.ObjectIDFromHex(trxnId)
			if err != nil {
				return err
			}
			filter := bson.M{"_id": oid}
			update := bson.M{"$set": bson.M{"in_plan": true}}
			_, err = h.Db.UpdateOne(h.C, filter, update)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// @Summary Get payment plans for a single user.
// @Description fetch all payment plans for the user by email.
// @Tags paymentplan
// @Param email path string true "User email"
// @Produce json
// @Success 200 {object} []models.PaymentPlan
// @Router /paymentplan/:email [get]
func GetPaymentPlans(h *Handler, planningUrl string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		email := c.Params("email")

		user, err := h.GetUserByEmail(email)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusNotFound, "error", "user not found", err.Error())
		}

		url := fmt.Sprintf("%s/payment_plans/%s", planningUrl, user.GetID().Hex())
		res, err := planningGetUserPaymentPlans(h, url)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "user payment plans not found", err.Error())
		}
		return FiberJsonResponse(c, fiber.StatusOK, "success", "user payment plans", res.PaymentPlans)
	}
}

type DeleteBody struct {
	TransactionIds []string `json:"transaction_ids,omitempty"`
}

// @Summary Delete a single PaymentPlan.
// @Description delete a single PaymentPlan by id.
// @Tags paymentplan
// @Param id path string true delete_body body DeleteBody "PaymentPlan Delete request"
// @Produce json
// @Success 200 {object} models.DeletePaymentPlanResponse
// @Router /paymentplan/:id [delete]
func DeletePaymentPlan(h *Handler, planningUrl string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		h.L.Info("This is body ", c.Body())
		if len(c.Body()) != 0 {
			var input DeleteBody
			if err := c.BodyParser(&input); err != nil {
				return FiberJsonResponse(c, fiber.StatusBadRequest, "error", "error parsing request ", err.Error())
			}
			h.L.Info("Delete Body", input)
			err := MarkTrxnAsNotPlanned(h, input.TransactionIds)
			if err != nil {
				return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "planning error failed to delete payment plan", err.Error())
			}
		}

		// get the id from the request params
		id := c.Params("id")
		url := fmt.Sprintf("%s/paymentplan/%s", planningUrl, id)
		res, err := planningDeletePaymentPlan(h, url)
		if err != nil {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "planning error failed to delete payment plan", err.Error())
		}
		if res.Status != models.DELETE_STATUS_SUCCESS {
			return FiberJsonResponse(c, fiber.StatusInternalServerError, "error", "delete payment plan status failed", res.Status)
		}

		return FiberJsonResponse(c, fiber.StatusOK, "success", "payment plan deleted", res)
	}
}

func MarkTrxnAsNotPlanned(h *Handler, transactionIds []string) error {
	oids := make([]primitive.ObjectID, len(transactionIds))
	for _, trxnId := range transactionIds {
		oid, err := primitive.ObjectIDFromHex(trxnId)
		if err != nil {
			return err
		}
		oids = append(oids, oid)
	}
	filter := bson.M{"_id": bson.M{"$in": oids}}
	update := bson.M{"$set": bson.M{"in_plan": false}}
	_, err := h.Db.UpdateMany(h.C, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func GetPaymentPlan(h *Handler, in *models.GetPaymentPlanRequest, planningUrl string) (*models.PaymentPlanResponse, error) {
	// create payment task from user inputs
	paymentTasks := make([]models.PaymentTask, len(in.AccountInfo))

	for idx, item := range in.AccountInfo {
		id, _ := primitive.ObjectIDFromHex(in.UserId)
		accId, _ := primitive.ObjectIDFromHex(item.AccountId)
		task := models.PaymentTask{
			UserId:       id,
			AccountId:    accId,
			Amount:       item.Amount,
			Transactions: item.TransactionIds,
		}
		paymentTasks[idx] = task
	}

	// save payment tasks to DB
	listOfIds, err := CreateManyPaymentTask(h, paymentTasks)
	if err != nil {
		h.L.Error("[PaymentTask] Error creating PaymentTasks ", "error", err, "paymentTasks", paymentTasks)
		return nil, err
	}

	for idx, id := range listOfIds {
		pt, _ := GetPaymentTask(h, id)
		paymentTasks[idx] = *pt
	}

	h.L.Info("PaymentTasks", paymentTasks)
	// send payment tasks to planning to get payment plans
	url := fmt.Sprintf("%s/paymentplan", planningUrl)
	res, err := planningCreatePaymentPlan(h, url, &models.CreatePaymentPlanRequest{PaymentTasks: paymentTasks, MetaData: in.MetaData, SavePlan: in.SavePlan})
	if err != nil {
		return nil, err
	}
	return &models.PaymentPlanResponse{PaymentPlans: res.PaymentPlans}, nil
}

func planningCreatePaymentPlan(h *Handler, url string, req *models.CreatePaymentPlanRequest) (*models.PaymentPlanResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := h.H.Post(url, "application/json", bytes.NewBuffer(body))
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

func planningDeletePaymentPlan(h *Handler, url string) (*models.DeletePaymentPlanResponse, error) {
	var body []byte

	r, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	resp, err := h.H.Do(r)
	if err != nil {
		panic(err)
	}

	var result models.DeletePaymentPlanResponse

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
		Transactions:     paymentTaskModel.Transactions,
	}
}
