package client

import (
	"encoding/json"
	"github.com/jalexanderII/zero-railway/models"
	"github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"os"
)

type TwilioClient struct {
	Client *twilio.RestClient
	L      *logrus.Logger
	number string
}

func NewTwilioClient(l *logrus.Logger) *TwilioClient {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioNumber := os.Getenv("TWILIO_PHONE_NUMBER")
	return &TwilioClient{
		Client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: accountSid,
			Password: authToken,
		}),
		L:      l,
		number: twilioNumber,
	}
}

func (t *TwilioClient) SendSMS(to, body string) (*models.SendSMSResponse, error) {
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(t.number)
	params.SetBody(body)

	resp, err := t.Client.Api.CreateMessage(params)
	if err != nil {
		t.L.Errorf("Error sending SMS: %s", err.Error())
		return &models.SendSMSResponse{Successful: false, ErrorMessage: err.Error()}, err
	} else {
		_, _ = json.Marshal(*resp)
		return &models.SendSMSResponse{Successful: true, ErrorMessage: "none"}, nil
	}
}
