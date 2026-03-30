package utils

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

func GenerateOTP() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func sendOTPViaSMS(phone, otp string) error {
	apiKey := os.Getenv("TWOFACTOR_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("TWOFACTOR_API_KEY not set")
	}

	url := fmt.Sprintf(
		"https://2factor.in/API/V1/%s/SMS/%s/%s/AUTOGEN",
		apiKey, phone, otp,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SMS send failed with status: %d body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func CreateOTP(phone string) error {
	otp := GenerateOTP()
	hash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	col := config.GetCollection("otps")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"phone": phone}
	update := bson.M{"$set": bson.M{"phone": phone, "otpHash": string(hash), "createdAt": time.Now()}}
	opts := options.Update().SetUpsert(true)

	_, err = col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	if os.Getenv("OTP_DEV_MODE") == "true" {
		fmt.Printf("\n[DEV] OTP for %s: %s\n\n", phone, otp)
		return nil
	}

	return sendOTPViaSMS(phone, otp)
}

func VerifyOTP(phone, otp string) (bool, error) {
	col := config.GetCollection("otps")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var record models.OTP
	err := col.FindOne(ctx, bson.M{"phone": phone}).Decode(&record)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check if OTP expired (5 minutes)
	if time.Since(record.CreatedAt) > 5*time.Minute {
		col.DeleteOne(ctx, bson.M{"phone": phone})
		return false, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(record.OTPHash), []byte(otp))
	if err != nil {
		return false, nil
	}

	// Delete used OTP
	col.DeleteOne(ctx, bson.M{"phone": phone})
	return true, nil
}
