package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/harentsoaR/dentist-api/internal/models"
)

// The service is now simpler, only handling SMS.
type NotificationService struct{}

// NewNotificationService is now much simpler.
func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// This function will now call the Textbelt API
func (s *NotificationService) SendAppointmentConfirmationSMS(patient *models.User, apt *models.Appointment) {
	if patient.Phone == "" {
		log.Println("SMS not sent: Patient has no phone number.")
		return
	}

	// The message for the SMS
	smsBody := fmt.Sprintf(
		"Appointment Confirmed: %s with %s on %s.",
		apt.Service,
		patient.FullName,
		apt.StartTime.Format("Jan 2 at 3:04 PM"),
	)

	// Send in a goroutine so it doesn't block the API response
	go sendSmsWithTextbelt(patient.Phone, smsBody)
}

// --- Private Helper Function for Textbelt ---
func sendSmsWithTextbelt(phone, message string) {
	// Textbelt free key allows 1 SMS per day. Get a paid key for more.
	// We'll get this from our .env file.
	textbeltKey := os.Getenv("TEXTBELT_API_KEY")

	postBody, _ := json.Marshal(map[string]string{
		"phone":   phone,
		"message": message,
		"key":     textbeltKey,
	})

	resp, err := http.Post("https://textbelt.com/text", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		log.Printf("Failed to send Textbelt request for number %s: %v", phone, err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	success, _ := result["success"].(bool)
	if !success {
		errorMsg, _ := result["error"].(string)
		log.Printf("Failed to send SMS via Textbelt to %s. Reason: %s", phone, errorMsg)
	} else {
		log.Printf("Successfully sent SMS via Textbelt to %s", phone)
	}
}
