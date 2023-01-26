package handlers

//func GetPaymentPlan(client core.CoreClient, ctx context.Context) func(c *fiber.Ctx) error {
//	return func(c *fiber.Ctx) error {
//		user, err := GetUserByEmail(client, ctx, c)
//		if err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on fetching user's from email", "data": err})
//		}
//
//		current_date := time.Now().Format("01.02.2006")
//		var input GetPaymentPlanRequest
//		if err := c.BodyParser(&input); err != nil {
//			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Error on login request", "data": err})
//		}
//		input.UserId = user.GetId()
//
//		accountInfoList := make([]*core.AccountInfo, len(input.AccountInfo))
//		for idx, accountInfo := range input.AccountInfo {
//			accountInfoList[idx] = AccountInfoDBToPB(accountInfo)
//		}
//		metaData := &common.MetaData{
//			PreferredPlanType:         common.PlanType(input.MetaData.PreferredPlanType),
//			PreferredTimelineInMonths: input.MetaData.PreferredTimelineInMonths,
//			PreferredPaymentFreq:      common.PaymentFrequency(input.MetaData.PreferredPaymentFreq),
//		}
//		paymentPlanResponse, err := client.GetPaymentPlan(ctx, &core.GetPaymentPlanRequest{AccountInfo: accountInfoList, UserId: input.UserId, MetaData: metaData, SavePlan: input.SavePlan})
//		if err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(err.Error())
//		}
//
//		responsePaymentPlans := make([]PaymentPlan, len(paymentPlanResponse.GetPaymentPlans()))
//		for idx, paymentPlan := range paymentPlanResponse.GetPaymentPlans() {
//			pp := CreateResponsePaymentPlan(paymentPlan)
//			name := fmt.Sprintf("Plan_%v_%v_%v", idx+1, pp.UserId[len(pp.UserId)-4:], current_date)
//			pp.Name = name
//			responsePaymentPlans[idx] = pp
//		}
//
//		return c.Status(fiber.StatusOK).JSON(responsePaymentPlans)
//	}
//}
