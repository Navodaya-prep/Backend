package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/navodayaprime/api/config"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var testMongoClient *mongo.Client

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", "handler-test-secret-key")
	os.Setenv("OTP_DEV_MODE", "true")

	uri := "mongodb://localhost:27017"
	if envURI := os.Getenv("MONGO_TEST_URI"); envURI != "" {
		uri = envURI
	}

	{
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		cancel()

		if err == nil {
			pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
			pingErr := client.Ping(pingCtx, nil)
			pingCancel()

			if pingErr == nil {
				config.DB = client.Database("navodaya_handlers_test")
				testMongoClient = client
			}
		}
	}

	code := m.Run()

	if testMongoClient != nil {
		config.DB.Drop(context.Background())
		testMongoClient.Disconnect(context.Background())
	}

	os.Exit(code)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func requireDB(t *testing.T) {
	t.Helper()
	if testMongoClient == nil {
		t.Skip("MongoDB not available — set MONGO_TEST_URI or start a local MongoDB")
	}
}

func dropCollection(t *testing.T, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	config.GetCollection(name).Drop(ctx)
}

// newRouter creates a gin router and registers a single handler at the given method+path.
func newRouter(method, path string, handlers ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Handle(method, path, handlers...)
	return r
}

// doRequest fires an HTTP request against the given router and returns the response.
func doRequest(r *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// parseBody parses the JSON body of a response.
func parseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to parse response body: %v\nbody: %s", err, w.Body.String())
	}
	return out
}

// setUserID injects userId and phone into the gin context (simulates RequireAuth middleware).
func setUserID(userID, phone string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userId", userID)
		c.Set("phone", phone)
		c.Next()
	}
}

// setAdminID injects admin context (simulates RequireAdmin middleware).
func setAdminID(adminID, email string, isSuper bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("adminId", adminID)
		c.Set("adminEmail", email)
		c.Set("isSuperAdmin", isSuper)
		c.Next()
	}
}
