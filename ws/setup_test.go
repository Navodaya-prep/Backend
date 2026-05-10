package ws

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
			config.DB = client.Database("navodaya_ws_test")
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

func requireDB(t *testing.T) {
	t.Helper()
	if testMongoClient == nil {
		t.Skip("MongoDB not available")
	}
}

func dropCollection(t *testing.T, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	config.GetCollection(name).Drop(ctx)
}
