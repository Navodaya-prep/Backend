package handlers

import (
	"net/http"

	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"
	"navodaya-api/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// LiveClassWS — GET /ws/live/:id
// Auth via query params:
//   - Students:  ?token=<jwt>&name=<userName>
//   - Teachers:  ?adminToken=<jwt>&name=<teacherName>
func LiveClassWS(c *gin.Context) {
	classID := c.Param("id")
	if _, err := primitive.ObjectIDFromHex(classID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class ID"})
		return
	}

	var userID, userName string
	var isTeacher bool

	adminToken := c.Query("adminToken")
	token := c.Query("token")
	name := c.Query("name")

	if adminToken != "" {
		// Teacher connection — validate admin JWT token
		adminClaims, err := utils.ParseAdminToken(adminToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin token"})
			return
		}
		isTeacher = true
		userID = adminClaims.AdminID
		userName = name
		if userName == "" {
			userName = adminClaims.Email // fallback to email
		}
	} else if token != "" {
		// Student connection — validate JWT
		claims, err := utils.ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID = claims.UserID

		// Fetch user name from DB
		oid, _ := primitive.ObjectIDFromHex(userID)
		ctx := c.Request.Context()
		var user models.User
		if err := config.GetCollection("users").FindOne(ctx, bson.M{"_id": oid}).Decode(&user); err == nil {
			userName = user.Name
		} else {
			userName = name // fallback to query param
		}
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token or adminToken required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &ws.Client{
		Hub:       ws.GlobalHub,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		ClassID:   classID,
		UserID:    userID,
		UserName:  userName,
		IsTeacher: isTeacher,
	}

	ws.GlobalHub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}
