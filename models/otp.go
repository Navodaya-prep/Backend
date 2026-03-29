package models

import "time"

type OTP struct {
	Phone     string    `bson:"phone"`
	OTPHash   string    `bson:"otpHash"`
	CreatedAt time.Time `bson:"createdAt"`
}
