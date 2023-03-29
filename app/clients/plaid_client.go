package client

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/jalexanderII/zero-railway/database"
	"github.com/jalexanderII/zero-railway/models"

	"github.com/plaid/plaid-go/plaid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var environments = map[string]plaid.Environment{
	"sandbox":     plaid.Sandbox,
	"development": plaid.Development,
	"production":  plaid.Production,
}

var environmentSecret = map[string]string{
	"sandbox":     "PLAID_SECRET_SANDBOX",
	"development": "PLAID_SECRET_DEV",
	"production":  "PLAID_SECRET_PROD",
}

var purposeToAccountFilter = map[models.Purpose]plaid.LinkTokenAccountFilters{
	models.PURPOSE_CREDIT: {Credit: &plaid.CreditFilter{AccountSubtypes: []plaid.AccountSubtype{plaid.ACCOUNTSUBTYPE_CREDIT_CARD}}},
	models.PURPOSE_DEBIT:  {Depository: &plaid.DepositoryFilter{AccountSubtypes: []plaid.AccountSubtype{plaid.ACCOUNTSUBTYPE_CHECKING}}},
}

type PlaidClient struct {
	// Name of the service
	Name string
	// Client is the object that contains all database functionalities
	Client       *plaid.PlaidApiService
	RedirectURL  string
	Products     []plaid.Products
	CountryCodes []plaid.CountryCode
	// custom logger
	L *logrus.Logger
	C context.Context
	// Database collection
	PlaidDb *mongo.Collection
	UserDb  *mongo.Collection
	AccDb   *mongo.Collection
	TrxnDb  *mongo.Collection
	// to pass tokens through methods
	LinkToken   *models.Token
	PublicToken *models.Token
}

func NewPlaidClient(collectionName string, l *logrus.Logger) *PlaidClient {
	// set constants from env
	PlaidDb := database.GetCollection(collectionName)
	UserDb := database.GetCollection(os.Getenv("USER_COLLECTION"))
	AccDb := database.GetCollection(os.Getenv("ACCOUNT_COLLECTION"))
	TrxnDb := database.GetCollection(os.Getenv("TRANSACTION_COLLECTION"))
	plaidEnv := os.Getenv("PLAID_ENV")
	plaidSecret := os.Getenv(environmentSecret[plaidEnv])
	plaidClient := os.Getenv("PLAID_CLIENT_ID")

	// create Plaid client
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", plaidClient)
	configuration.AddDefaultHeader("PLAID-SECRET", plaidSecret)
	configuration.UseEnvironment(environments[plaidEnv])

	countryCodes := convertCountryCodes(strings.Split(os.Getenv("PLAID_COUNTRY_CODES"), ","))
	products := convertProducts(strings.Split(os.Getenv("PLAID_PRODUCTS"), ","))
	client := plaid.NewAPIClient(configuration)
	return &PlaidClient{
		Name:         "ZeroFintech",
		Client:       client.PlaidApi,
		RedirectURL:  os.Getenv("PLAID_REDIRECT_URI"),
		Products:     products,
		CountryCodes: countryCodes,
		L:            l,
		C:            context.Background(),
		PlaidDb:      PlaidDb,
		UserDb:       UserDb,
		AccDb:        AccDb,
		TrxnDb:       TrxnDb,
		LinkToken:    nil,
		PublicToken:  nil,
	}
}

// LinkTokenCreate creates a link token using the specified parameters
func (p *PlaidClient) LinkTokenCreate(email, purpose string) (*models.CreateLinkTokenResponse, error) {
	fmt.Printf("email: %+v", email)
	fmt.Printf(" purpose: %+v", purpose)

	purp, err := models.PurposeFromString(purpose)
	if err != nil {
		return nil, err
	}

	DbUser, err := p.GetUser(email)
	if err != nil {
		p.L.Error("[DB Error] error fetching user", err)
		return nil, err
	}
	id := DbUser.ID.Hex()

	user := plaid.LinkTokenCreateRequestUser{
		ClientUserId: id,
	}
	request := plaid.NewLinkTokenCreateRequest(p.Name, "en", p.CountryCodes, user)
	request.SetRedirectUri(p.RedirectURL)

	p.L.Infof("The link purpose is %+v", purp)
	if purp == models.PURPOSE_DEBIT {
		p.Products = convertProducts([]string{"transactions"})
	}

	request.SetProducts(p.Products)
	request.SetAccountFilters(purposeToAccountFilter[purp])

	p.L.Infof("Link token request %+v", request)
	linkTokenCreateResp, _, err := p.Client.LinkTokenCreate(p.C).LinkTokenCreateRequest(*request).Execute()
	if err != nil {
		p.L.Errorf("[Plaid Error] error creating link token %+v", renderError(err)["error"])
		return nil, err
	}

	p.L.Info("link token created: ", linkTokenCreateResp)
	return &models.CreateLinkTokenResponse{Token: linkTokenCreateResp.GetLinkToken(), UserId: id}, nil
}

// ExchangePublicToken this function takes care of creating the permanent access token
// that will be stored in the database for cross-platform connection to users' bank.
// If for whatever reason there is a problem with the client or public token, there
// are json responses and logs that will adequately reflect all issues
func (p *PlaidClient) ExchangePublicToken(ctx context.Context, publicToken string) (*models.Token, error) {
	// exchange the public_token for an access_token
	exchangePublicTokenResp, _, err := p.Client.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(
		*plaid.NewItemPublicTokenExchangeRequest(publicToken),
	).Execute()
	if err != nil {
		p.L.Errorf("[Plaid Error] error getting exchangePublicTokenResp %+v", renderError(err)["error"])
		return nil, err
	}

	accessToken := exchangePublicTokenResp.GetAccessToken()
	itemID := exchangePublicTokenResp.GetItemId()
	if itemExists(p.Products, plaid.PRODUCTS_TRANSFER) {
		_, err = p.authorizeAndCreateTransfer(ctx, accessToken)
		if err != nil {
			p.L.Errorf("[Plaid Error] error authorizeAndCreateTransfer %+v", renderError(err)["error"])
			return nil, err
		}
	}

	p.L.Info("public token: " + publicToken)
	p.L.Info("access token: " + accessToken)
	p.L.Info("item ID: " + itemID)
	return &models.Token{Value: accessToken, ItemId: itemID}, nil
}

func (p *PlaidClient) GetAccountDetails(token *models.Token) (*models.AccountDetailsResponse, error) {
	var liabilitiesResponse models.LiabilitiesResponse
	var transactionsResponse models.TransactionsResponse

	if token.Purpose == models.PURPOSE_DEBIT {
		// if debit get account info only
		creditAccounts, creditTransactions, err := p.fetchDebitInfo(token.Value)
		if err != nil {
			return nil, err
		}
		transactionsResponse = models.TransactionsResponse{Accounts: creditAccounts, Transactions: creditTransactions}
	} else {
		// otherwise use liabilities request to get credit card accounts and also fetch transactions
		liabilitiesReq := plaid.NewLiabilitiesGetRequest(token.Value)
		liabilitiesResp, _, err := p.Client.LiabilitiesGet(p.C).LiabilitiesGetRequest(*liabilitiesReq).Execute()
		if err != nil {
			p.L.Errorf("[Plaid Error] getting Liabilities %+v", renderError(err)["error"])
			return nil, err
		}
		liabilitiesResponse = models.LiabilitiesResponse{Liabilities: liabilitiesResp.GetLiabilities().Credit}

		creditAccounts, creditTransactions, err := p.fetchCreditInfo(token.Value, true)
		if err != nil {
			return nil, err
		}
		transactionsResponse = models.TransactionsResponse{Accounts: creditAccounts, Transactions: creditTransactions}
	}

	response, err := p.PlaidResponseToPB(liabilitiesResponse, transactionsResponse, token.User, token.Purpose)
	if err != nil {
		p.L.Error("Error converting PlaidResponse to PB", "error", err)
		return nil, err
	}
	return response, nil
}

func (p *PlaidClient) fetchDebitInfo(accessToken string) ([]plaid.AccountBase, []plaid.Transaction, error) {
	// if debit get account info only
	accountsReq := plaid.NewAccountsGetRequest(accessToken)
	accountsResp, _, err := p.Client.AccountsGet(p.C).AccountsGetRequest(*accountsReq).Execute()
	if err != nil {
		p.L.Errorf("[Plaid Error] getting Liabilities %+v", renderError(err)["error"])
		return nil, nil, err
	}

	var debitAccounts []plaid.AccountBase
	for _, account := range accountsResp.GetAccounts() {
		if account.Type == plaid.ACCOUNTTYPE_DEPOSITORY {
			debitAccounts = append(debitAccounts, account)
		}
	}
	return debitAccounts, nil, nil
}

func (p *PlaidClient) fetchCreditInfo(accessToken string, fetchAll bool) ([]plaid.AccountBase, []plaid.Transaction, error) {
	var creditAccounts []plaid.AccountBase
	var creditTransactions []plaid.Transaction
	accountIds := make(map[string]string)
	trxnIds := make(map[string]bool)

	const iso8601TimeFormat = "2006-01-02"
	endDate := time.Now().Local().Format(iso8601TimeFormat)
	// Last Quarter (90 days)
	startDate := time.Now().Local().Add(-90 * 24 * time.Hour).Format(iso8601TimeFormat)

	request := plaid.NewTransactionsGetRequest(
		accessToken,
		startDate,
		endDate,
	)
	fetchNext := int32(500)
	// last 350 transactions
	request.SetOptions(plaid.TransactionsGetRequestOptions{Count: plaid.PtrInt32(fetchNext)})

	transactionsResp, _, err := p.Client.TransactionsGet(p.C).TransactionsGetRequest(*request).Execute()
	if err != nil {
		p.L.Errorf("[Plaid Error] getting Transactions %+v", renderError(err)["error"])
		return nil, nil, err
	}

	for _, account := range transactionsResp.GetAccounts() {
		if account.Type == plaid.ACCOUNTTYPE_CREDIT {
			if _, ok := accountIds[account.AccountId]; !ok {
				creditAccounts = append(creditAccounts, account)
				accountIds[account.AccountId] = account.Name
			}
		}
	}

	for _, transaction := range transactionsResp.GetTransactions() {
		if _, ok := accountIds[transaction.AccountId]; ok {
			if _, ok := trxnIds[transaction.TransactionId]; !ok {
				creditTransactions = append(creditTransactions, transaction)
				trxnIds[transaction.TransactionId] = true
			}
		}
	}

	totalNumberOfTransactions := transactionsResp.GetTotalTransactions()
	if totalNumberOfTransactions > fetchNext && fetchAll {
		p.L.Info("Fetching all transactions, there are more than ", fetchNext)
		// if there are more than 500 transactions, fetch the rest
		for i := fetchNext; i < totalNumberOfTransactions; i += int32(math.Min(float64(fetchNext), float64(totalNumberOfTransactions-i))) {
			request.SetOptions(plaid.TransactionsGetRequestOptions{Count: plaid.PtrInt32(fetchNext), Offset: plaid.PtrInt32(i)})
			transactionsResp, _, err = p.Client.TransactionsGet(p.C).TransactionsGetRequest(*request).Execute()
			if err != nil {
				p.L.Errorf("[Plaid Error] getting Transactions %+v", renderError(err)["error"])
				return nil, nil, err
			}
			for _, account := range transactionsResp.GetAccounts() {
				if account.Type == plaid.ACCOUNTTYPE_CREDIT {
					if _, ok := accountIds[account.AccountId]; !ok {
						creditAccounts = append(creditAccounts, account)
						accountIds[account.AccountId] = account.Name
					}
				}
			}

			for _, transaction := range transactionsResp.GetTransactions() {
				if _, ok := accountIds[transaction.AccountId]; ok {
					if _, ok := trxnIds[transaction.TransactionId]; !ok {
						creditTransactions = append(creditTransactions, transaction)
						trxnIds[transaction.TransactionId] = true
					}
				}
			}
		}
	}
	return creditAccounts, creditTransactions, nil
}

func (p *PlaidClient) PlaidResponseToPB(lr models.LiabilitiesResponse, tr models.TransactionsResponse, user *models.User, purpose models.Purpose) (*models.AccountDetailsResponse, error) {
	UserId := user.ID.Hex()

	accountLiabilities := make(map[string]plaid.CreditCardLiability)
	for _, al := range lr.Liabilities {
		if al.AccountId.IsSet() {
			accId := al.AccountId.Get()
			accountLiabilities[*accId] = al
		} else {
			p.L.Error("Error isolating accountLiabilities")
			return nil, errors.New("error isolating accountLiabilities")
		}
	}

	accounts := make([]*models.Account, len(tr.Accounts))
	if purpose == models.PURPOSE_DEBIT {
		for idx, account := range tr.Accounts {
			userId, err := primitive.ObjectIDFromHex(UserId)
			if err != nil {
				userId = primitive.NewObjectID()
			}
			accounts[idx] = &models.Account{
				UserId:           userId,
				Name:             account.Name,
				OfficialName:     account.GetOfficialName(),
				Type:             string(account.Type),
				Subtype:          string(account.GetSubtype()),
				AvailableBalance: float64(account.Balances.GetAvailable()),
				CurrentBalance:   float64(account.Balances.GetCurrent()),
				IsoCurrencyCode:  account.Balances.GetIsoCurrencyCode(),
				PlaidAccountId:   account.AccountId,
			}
		}

	} else {
		for idx, account := range tr.Accounts {
			if acc, ok := accountLiabilities[account.AccountId]; ok {
				aprs := make([]*models.AnnualPercentageRates, len(acc.Aprs))
				for x, apr := range acc.Aprs {
					aprs[x] = &models.AnnualPercentageRates{
						AprPercentage:        float64(apr.AprPercentage),
						AprType:              apr.AprType,
						BalanceSubjectToApr:  float64(apr.GetBalanceSubjectToApr()),
						InterestChargeAmount: float64(apr.GetInterestChargeAmount()),
					}
				}
				userId, err := primitive.ObjectIDFromHex(UserId)
				if err != nil {
					userId = primitive.NewObjectID()
				}
				accounts[idx] = &models.Account{
					ID:                     account.AccountId,
					UserId:                 userId,
					Name:                   account.Name,
					OfficialName:           account.GetOfficialName(),
					Type:                   string(account.Type),
					Subtype:                string(account.GetSubtype()),
					AvailableBalance:       float64(account.Balances.GetAvailable()),
					CurrentBalance:         float64(account.Balances.GetCurrent()),
					CreditLimit:            float64(account.Balances.GetLimit()),
					IsoCurrencyCode:        account.Balances.GetIsoCurrencyCode(),
					AnnualPercentageRate:   aprs,
					IsOverdue:              acc.GetIsOverdue(),
					LastPaymentAmount:      float64(acc.LastPaymentAmount),
					LastStatementIssueDate: acc.LastStatementIssueDate,
					LastStatementBalance:   float64(acc.LastStatementBalance),
					MinimumPaymentAmount:   float64(acc.MinimumPaymentAmount),
					NextPaymentDueDate:     acc.GetNextPaymentDueDate(),
					PlaidAccountId:         account.AccountId,
				}
			}
		}
	}
	var transactions []*models.Transaction
	for _, transaction := range tr.Transactions {
		userId, err := primitive.ObjectIDFromHex(UserId)
		if err != nil {
			userId = primitive.NewObjectID()
		}
		timeLayout := "2006-01-02" // This defines the layout pattern, not an actual date
		// Parse ISO 8601 date string to Go time.Time
		parsedDate, err := time.Parse(timeLayout, transaction.Date)
		if err != nil {
			return nil, fmt.Errorf("error parsing transaction Date. transaction.Date: %s. Error: %s", transaction.Date, err.Error())
		}
		// Get Unix timestamp in milliseconds
		unixTimestampMillis := parsedDate.UnixNano() / int64(time.Millisecond)
		transactions = append(transactions, &models.Transaction{
			ID:                   transaction.TransactionId,
			UserId:               userId,
			TransactionType:      transaction.GetTransactionType(),
			PendingTransactionId: transaction.GetPendingTransactionId(),
			CategoryId:           transaction.GetCategoryId(),
			Category:             transaction.Category,
			TransactionDetails: &models.TransactionDetails{
				Address:         transaction.Location.GetAddress(),
				City:            transaction.Location.GetCity(),
				State:           transaction.Location.GetRegion(),
				Zipcode:         transaction.Location.GetPostalCode(),
				Country:         transaction.Location.GetCountry(),
				StoreNumber:     transaction.Location.GetStoreNumber(),
				ReferenceNumber: transaction.PaymentMeta.GetReferenceNumber(),
			},
			Name:                transaction.Name,
			OriginalDescription: transaction.GetOriginalDescription(),
			Amount:              float64(transaction.Amount),
			IsoCurrencyCode:     transaction.GetIsoCurrencyCode(),
			Date:                unixTimestampMillis,
			Pending:             transaction.Pending,
			MerchantName:        transaction.GetMerchantName(),
			PaymentChannel:      transaction.PaymentChannel,
			AuthorizedDate:      transaction.GetAuthorizedDate(),
			PrimaryCategory:     transaction.GetPersonalFinanceCategory().Primary,
			DetailedCategory:    transaction.GetPersonalFinanceCategory().Detailed,
			PlaidAccountId:      transaction.AccountId,
			PlaidTransactionId:  transaction.TransactionId,
			InPlan:              false,
		})
	}
	return &models.AccountDetailsResponse{
		Accounts:     accounts,
		Transactions: transactions,
	}, nil
}

// SaveToken method adds the permanent plaid token and stores into the plaid tokens' table with the
// same id as the user.
func (p *PlaidClient) SaveToken(token *models.Token) error {
	token.ID = primitive.NewObjectID()
	_, err := p.PlaidDb.InsertOne(p.C, token)
	if err != nil {
		p.L.Info("Error inserting new Token ", err)
		return err
	}
	return nil
}

func (p *PlaidClient) UpdateToken(TokenId primitive.ObjectID, value, itemId string) error {
	filter := bson.D{{Key: "_id", Value: TokenId}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "value", Value: value}, {Key: "item_id", Value: itemId}}}}
	_, err := p.PlaidDb.UpdateOne(p.C, filter, update)
	if err != nil {
		return err
	}
	return nil
}

// GetTokens returns every token associated to the user in the form of a slice of Token pointers.
func (p *PlaidClient) GetTokens(Id primitive.ObjectID) (*[]models.Token, error) {
	var results []models.Token
	cursor, err := p.PlaidDb.Find(p.C, bson.D{{Key: "user._id", Value: Id}})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(p.C, &results); err != nil {
		p.L.Error("[PlaidDb] Error getting all users tokens", "error", err)
		return nil, err
	}
	return &results, nil
}

// GetToken will get a token from the database and return it given the user's ID and the
// token id
func (p *PlaidClient) GetToken(accessToken, tokenId string) (*models.Token, error) {
	var token models.Token
	var filter []bson.M

	if tokenId != "" {
		id, err := primitive.ObjectIDFromHex(tokenId)
		if err != nil {
			return nil, err
		}
		filter = []bson.M{{"_id": id}, {"value": accessToken}}
	} else {
		filter = []bson.M{{"value": accessToken}}
	}

	err := p.PlaidDb.FindOne(p.C, bson.M{"$or": filter}).Decode(&token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (p *PlaidClient) GetUserToken(user *models.User) (*models.Token, error) {
	var token models.Token
	filter := []bson.M{{"user._id": user.ID}, {"user.username": user.Username}, {"user.email": user.Email}}
	err := p.PlaidDb.FindOne(p.C, bson.M{"$or": filter}).Decode(&token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (p *PlaidClient) GetUser(email string) (*models.User, error) {
	user, err := p.GetUserByEmail(email)
	if err != nil {
		fmt.Printf("failed to get a user: %+v", email)
		return nil, err
	}
	return &models.User{
		ID:       *user.GetID(),
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

func (p *PlaidClient) GetUserByClarkId(userId string) (*models.User, error) {
	var user models.User
	filter := bson.M{"clerk_id": userId}
	err := p.UserDb.FindOne(p.C, filter).Decode(&user)
	if err != nil {
		p.L.Errorf("failed to get a user: %s", userId)
		return nil, err
	}
	return &models.User{
		ID:       *user.GetID(),
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

func (p *PlaidClient) SetLinkToken(token *models.Token) {
	p.LinkToken = token
}

func (p *PlaidClient) SetPublicToken(token *models.Token) {
	p.PublicToken = token
}

func (p *PlaidClient) GetLinkToken() *models.Token {
	return p.LinkToken
}

func (p *PlaidClient) GetPublicToken() *models.Token {
	return p.PublicToken
}

func convertCountryCodes(countryCodeStrs []string) []plaid.CountryCode {
	var countryCodes []plaid.CountryCode

	for _, countryCodeStr := range countryCodeStrs {
		countryCodes = append(countryCodes, plaid.CountryCode(countryCodeStr))
	}

	return countryCodes
}

func convertProducts(productStrs []string) []plaid.Products {
	var products []plaid.Products

	for _, productStr := range productStrs {
		products = append(products, plaid.Products(productStr))
	}

	return products
}

func renderError(originalErr error) map[string]interface{} {
	resp := make(map[string]interface{})
	if plaidError, err := plaid.ToPlaidError(originalErr); err == nil {
		resp["error"] = plaidError
		return resp
	}
	resp["error"] = originalErr.Error()
	return resp
}

// This is a helper function to authorize and create a Transfer after successful
// exchange of a public_token for an access_token. The transfer_id is then used
// to obtain the data about that particular Transfer.
func (p *PlaidClient) authorizeAndCreateTransfer(ctx context.Context, accessToken string) (string, error) {
	// We call /accounts/get to obtain first account_id - in production,
	// account_id's should be persisted in a data store and retrieved
	// from there.
	accountsGetResp, _, _ := p.Client.AccountsGet(ctx).AccountsGetRequest(
		*plaid.NewAccountsGetRequest(accessToken),
	).Execute()

	accountID := accountsGetResp.GetAccounts()[0].AccountId

	transferAuthorizationCreateUser := plaid.NewTransferUserInRequest("FirstName LastName")
	transferAuthorizationCreateRequest := plaid.NewTransferAuthorizationCreateRequest(
		accessToken,
		accountID,
		"credit",
		"ach",
		"1.34",
		"ppd",
		*transferAuthorizationCreateUser,
	)
	transferAuthorizationCreateResp, _, err := p.Client.TransferAuthorizationCreate(ctx).TransferAuthorizationCreateRequest(*transferAuthorizationCreateRequest).Execute()
	if err != nil {
		return "", err
	}
	authorizationID := transferAuthorizationCreateResp.GetAuthorization().Id

	transferCreateRequest := plaid.NewTransferCreateRequest(
		"key",
		accessToken,
		accountID,
		authorizationID,
		"credit",
		"ach",
		"1.34",
		"Payment",
		"ppd",
		*transferAuthorizationCreateUser,
	)
	transferCreateResp, _, err := p.Client.TransferCreate(ctx).TransferCreateRequest(*transferCreateRequest).Execute()
	if err != nil {
		return "", err
	}

	return transferCreateResp.GetTransfer().Id, nil
}

// Helper function to determine if Transfer is in Plaid product array
func itemExists(array []plaid.Products, product plaid.Products) bool {
	for _, item := range array {
		if item == product {
			return true
		}
	}

	return false
}

func (p *PlaidClient) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	filter := bson.M{"email": email}
	err := p.UserDb.FindOne(p.C, filter).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
