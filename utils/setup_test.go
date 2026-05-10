package utils

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/navodayaprime/api/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var testMongoClient *mongo.Client

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	os.Setenv("OTP_DEV_MODE", "true")

	uri := "mongodb://localhost:27017"
	if envURI := os.Getenv("MONGO_TEST_URI"); envURI != "" {
		uri = envURI
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	cancel()

	if err == nil {
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
		pingErr := client.Ping(pingCtx, nil)
		pingCancel()

		if pingErr == nil {
			config.DB = client.Database("navodaya_utils_test")
			testMongoClient = client
		}
	}

	code := m.Run()

	if testMongoClient != nil {
		config.DB.Drop(context.Background())
		testMongoClient.Disconnect(context.Background())
	}

	os.Exit(code)
}

func dbAvailable() bool {
	return testMongoClient != nil
}

func requireDB(t *testing.T) {
	t.Helper()
	if !dbAvailable() {
		t.Skip("MongoDB not available — set MONGO_TEST_URI or start a local MongoDB")
	}
}

func clearCollection(t *testing.T, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	config.GetCollection(name).Drop(ctx)
}
