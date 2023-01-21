package database

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jalexanderII/zero-railway/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const Performance = 100

var (
	// Used to create a singleton object of MongoDB client.
	// Initialized and exposed through  GetMongoClient()
	mongoClient *mongo.Client
	// Used during creation of singleton client object in GetMongoClient()
	clientInstanceError error
	// Used to execute client creation procedure only once
	mongoOnce sync.Once
	dbName    string
	ctx       context.Context
)

func GetCollection(name string) *mongo.Collection {
	return mongoClient.Database(dbName).Collection(name)
}

func StartMongoDB() error {
	uri := config.GetEnv("MONGODB_URI")
	if uri == "" {
		return errors.New("you must set your 'MONGODB_URI' environmental variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
	}

	database := config.GetEnv("DATABASE")
	if database == "" {
		return errors.New("you must set your 'DATABASE' environmental variable")
	} else {
		dbName = database
	}

	// Perform connection creation operation only once.
	mongoOnce.Do(func() {
		// Set client options
		clientOptions := options.Client().ApplyURI(uri)
		var cancel context.CancelFunc
		ctx, cancel = NewDBContext(10 * time.Second)
		defer cancel()
		// Connect to MongoDB
		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			clientInstanceError = err
		}
		// Check the connection
		err = client.Ping(ctx, nil)
		if err != nil {
			clientInstanceError = err
		}
		mongoClient = client
	})

	if clientInstanceError != nil {
		panic(clientInstanceError)
	}
	return nil
}

func CloseMongoDB() {
	err := mongoClient.Disconnect(ctx)
	if err != nil {
		panic(err)
	}
}

// NewDBContext returns a new Context according to app performance
func NewDBContext(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d*Performance/100)
}
