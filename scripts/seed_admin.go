package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type Admin struct {
	FirstName    string    `bson:"firstName"`
	LastName     string    `bson:"lastName"`
	Email        string    `bson:"email"`
	Password     string    `bson:"password"`
	IsSuperAdmin bool      `bson:"isSuperAdmin"`
	IsActive     bool      `bson:"isActive"`
	CreatedAt    time.Time `bson:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt"`
}

func main() {
	// Get MongoDB URI from environment variable
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017" // Default local MongoDB
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Check connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	fmt.Println("✅ Connected to MongoDB")

	// Get database and collection
	db := client.Database("navodaya_prime")
	adminsCollection := db.Collection("admins")

	// Check if admin already exists
	email := "admin@navodaya.com"
	existingAdmin := adminsCollection.FindOne(ctx, bson.M{"email": email})
	if existingAdmin.Err() == nil {
		fmt.Println("⚠️  Super admin already exists with email:", email)
		fmt.Println("To reset password, use MongoDB shell or update manually")
		return
	}

	// Hash the default password
	password := "admin123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Create super admin
	admin := Admin{
		FirstName:    "Super",
		LastName:     "Admin",
		Email:        email,
		Password:     string(hashedPassword),
		IsSuperAdmin: true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	result, err := adminsCollection.InsertOne(ctx, admin)
	if err != nil {
		log.Fatalf("Failed to create admin: %v", err)
	}

	fmt.Println("✅ Super admin created successfully!")
	fmt.Println("   ID:", result.InsertedID)
	fmt.Println("   Email:", email)
	fmt.Println("   Password:", password)
	fmt.Println("")
	fmt.Println("⚠️  IMPORTANT: Change this password after first login!")
	fmt.Println("")
	fmt.Println("You can now login to the admin panel at:")
	fmt.Println("   http://localhost:5173 (development)")
	fmt.Println("")
}
