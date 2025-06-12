package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/harentsoaR/dentist-api/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- CREATE APPOINTMENT (Enhanced with Notifications) ---
func (h *Handler) CreateAppointment(c *gin.Context) {
	var req struct {
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
		Service   string `json:"service"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	startTime, err1 := time.Parse(time.RFC3339, req.StartTime)
	endTime, err2 := time.Parse(time.RFC3339, req.EndTime)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time format, use RFC3339"})
		return
	}

	userIDHex, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")
	if userRole != "client" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only clients can book appointments."})
		return
	}

	patientID, _ := primitive.ObjectIDFromHex(userIDHex.(string))

	// Get full patient details for notifications
	var patient models.User
	userCollection := h.DB.Collection("users")
	err := userCollection.FindOne(context.TODO(), bson.M{"_id": patientID}).Decode(&patient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not find user details"})
		return
	}

	apt := models.Appointment{
		ID:          primitive.NewObjectID(),
		PatientID:   patientID,
		PatientName: patient.FullName,
		StartTime:   startTime,
		EndTime:     endTime,
		Service:     req.Service,
		Status:      "Scheduled", // Set default status
	}

	collection := h.DB.Collection("appointments")
	_, err = collection.InsertOne(context.TODO(), apt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create appointment"})
		return
	}

	// --- NOTIFICATION ---
	h.NotificationSvc.SendAppointmentConfirmationSMS(&patient, &apt)

	c.JSON(http.StatusCreated, apt)
}

// --- GET APPOINTMENTS (with Filtering & Sorting) ---
func (h *Handler) GetAppointments(c *gin.Context) {
	filter := bson.M{}

	// Filter by date range (e.g., /api/appointments?startDate=2024-07-01&endDate=2024-07-31)
	if startDateStr := c.Query("startDate"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			filter["startTime"] = bson.M{"$gte": startDate}
		}
	}
	if endDateStr := c.Query("endDate"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Add 24 hours to include the entire end day
			endDate = endDate.Add(23*time.Hour + 59*time.Minute)
			if f, ok := filter["startTime"].(bson.M); ok {
				f["$lte"] = endDate
			} else {
				filter["startTime"] = bson.M{"$lte": endDate}
			}
		}
	}

	// Filter by status (e.g., /api/appointments?status=Scheduled)
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}

	// Sort by start time to "group" by date
	findOptions := options.Find().SetSort(bson.D{{Key: "startTime", Value: 1}}) // 1 for ascending

	collection := h.DB.Collection("appointments")
	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve appointments"})
		return
	}
	defer cursor.Close(context.TODO())

	var appointments []models.Appointment
	if err = cursor.All(context.TODO(), &appointments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode appointments"})
		return
	}

	c.JSON(http.StatusOK, appointments)
}

// --- GET APPOINTMENTS FOR A USER (with Role-Based Filtering) ---
func (h *Handler) GetAppointment(c *gin.Context) {
	// --- SECURITY & ROLE-BASED LOGIC ---
	// Get user info from the context (set by your JWT middleware)
	userIDHex, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")

	// Start with an empty filter
	filter := bson.M{}

	// If the user is a 'client', force the filter to only include their appointments
	if userRole == "client" {
		patientID, err := primitive.ObjectIDFromHex(userIDHex.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID in token"})
			return
		}
		filter["patientId"] = patientID
	}
	// If the user is a 'dentist' or 'staff', the filter remains empty, so they can see all appointments.
	// We can add more specific filters for them later if needed (e.g., filter by a specific patient ID from a query param).

	// --- EXISTING FILTERING LOGIC (Can be combined with the role-based filter) ---

	// Filter by date range (e.g., /api/appointments?startDate=2024-07-01&endDate=2024-07-31)
	if startDateStr := c.Query("startDate"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			filter["startTime"] = bson.M{"$gte": startDate}
		}
	}
	if endDateStr := c.Query("endDate"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Add time to include the entire end day
			endDate = endDate.Add(23*time.Hour + 59*time.Minute)
			if f, ok := filter["startTime"].(bson.M); ok {
				f["$lte"] = endDate
			} else {
				filter["startTime"] = bson.M{"$lte": endDate}
			}
		}
	}

	// Filter by status (e.g., /api/appointments?status=Scheduled)
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}

	// For staff/dentists who might want to look up a specific patient's appointments
	if userRole != "client" {
		if patientIDQuery := c.Query("patientId"); patientIDQuery != "" {
			pID, err := primitive.ObjectIDFromHex(patientIDQuery)
			if err == nil {
				filter["patientId"] = pID
			}
		}
	}

	// Sort by start time to "group" by date
	findOptions := options.Find().SetSort(bson.D{{Key: "startTime", Value: -1}}) // -1 for descending (newest first)

	collection := h.DB.Collection("appointments")
	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve appointments"})
		return
	}
	defer cursor.Close(context.TODO())

	var appointments []models.Appointment
	if err = cursor.All(context.TODO(), &appointments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode appointments"})
		return
	}

	// It's good practice to return an empty array instead of `nil` if no appointments are found
	if appointments == nil {
		appointments = make([]models.Appointment, 0)
	}

	c.JSON(http.StatusOK, appointments)
}

// --- UPDATE APPOINTMENT (Dentist/Staff Only) ---
func (h *Handler) UpdateAppointment(c *gin.Context) {
	userRole, _ := c.Get("userRole")
	if userRole != "dentist" && userRole != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied."})
		return
	}

	appointmentID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid appointment ID"})
		return
	}

	var req struct {
		StartTime *string `json:"startTime,omitempty"`
		EndTime   *string `json:"endTime,omitempty"`
		Service   *string `json:"service,omitempty"`
		Status    *string `json:"status,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updateFields := bson.M{}
	if req.StartTime != nil {
		if t, err := time.Parse(time.RFC3339, *req.StartTime); err == nil {
			updateFields["startTime"] = t
		}
	}
	if req.EndTime != nil {
		if t, err := time.Parse(time.RFC3339, *req.EndTime); err == nil {
			updateFields["endTime"] = t
		}
	}
	if req.Service != nil {
		updateFields["service"] = *req.Service
	}
	if req.Status != nil {
		updateFields["status"] = *req.Status
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	collection := h.DB.Collection("appointments")
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": appointmentID}, bson.M{"$set": updateFields})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update appointment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appointment updated successfully"})
}

// --- CANCEL APPOINTMENT (Dentist/Staff Only) ---
func (h *Handler) CancelAppointment(c *gin.Context) {
	userRole, _ := c.Get("userRole")
	if userRole != "dentist" && userRole != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied."})
		return
	}

	appointmentID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid appointment ID"})
		return
	}

	collection := h.DB.Collection("appointments")

	// Find the appointment first to get patient info for notification
	var apt models.Appointment
	err = collection.FindOne(context.TODO(), bson.M{"_id": appointmentID}).Decode(&apt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Appointment not found"})
		return
	}

	// Update the status to "Cancelled"
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": appointmentID}, bson.M{"$set": bson.M{"status": "Cancelled"}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel appointment"})
		return
	}

	// Find patient details for notification
	var patient models.User
	userCollection := h.DB.Collection("users")
	err = userCollection.FindOne(context.TODO(), bson.M{"_id": apt.PatientID}).Decode(&patient)
	if err == nil {
		apt.Status = "Cancelled" // Manually update status for the notification object
		h.NotificationSvc.SendAppointmentConfirmationSMS(&patient, &apt)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appointment cancelled successfully"})
}
