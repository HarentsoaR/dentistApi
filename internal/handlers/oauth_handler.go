package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"github.com/harentsoaR/dentist-api/internal/config"
	"github.com/harentsoaR/dentist-api/internal/services"
)

// OAuthHandler handles OAuth-related endpoints
type OAuthHandler struct {
	oauthService *services.OAuthService
}

// NewOAuthHandler creates a new OAuthHandler
func NewOAuthHandler(oauthService *services.OAuthService) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
	}
}

// InitGoogleLogin starts the Google OAuth login process
func (h *OAuthHandler) InitGoogleLogin(c *gin.Context) {
	// Generating a secure random state
	state, err := config.GenerateRandomState()
	if err != nil {
		log.Printf("Failed to generate state: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	// Storing the state in the session
	session := sessions.Default(c)
	session.Options(sessions.Options{
		Path:     "/",
		MaxAge:   60 * 15, // 15 minutes
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production",
	})
	
	session.Set("oauth_state", state)
	if err := session.Save(); err != nil {
		log.Printf("Failed to save session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
		return
	}

	log.Printf("Generated OAuth state: %s", state)
	
	// Generating the OAuth URL with just the state and access type
	url := config.GoogleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	log.Printf("Redirecting to Google OAuth: %s", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// HandleGoogleCallback processes the callback from Google after user authentication
func (h *OAuthHandler) HandleGoogleCallback(c *gin.Context) {
	// Get the state and code from the query
	state := c.Query("state")
	code := c.Query("code")

	log.Printf("OAuth callback received. State: %s, Code: %s", state, code)

	// Get the session
	session := sessions.Default(c)
	session.Options(sessions.Options{
		Path:     "/",
		MaxAge:   60 * 15, // 15 minutes
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production",
	})


	// Get the state from session
	sessionState, ok := session.Get("oauth_state").(string)
	if !ok || sessionState == "" {
		log.Printf("No state found in session")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter: no state in session"})
		return
	}

	// Clear the state from session immediately after retrieving it
	session.Delete("oauth_state")
	if err := session.Save(); err != nil {
		log.Printf("Failed to clear session state: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear session state"})
		return
	}

	// Verify state parameter matches the one from session
	if state == "" || state != sessionState {
		log.Printf("Invalid state: state='%s', sessionState='%s' (type: %T)", state, sessionState, sessionState)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter: state mismatch"})
		return
	}

	// Clear the state from session
	session.Delete("oauth_state")
	if err := session.Save(); err != nil {
		log.Printf("Failed to clear session state: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear session"})
		return
	}

	// Verify code is present
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization code not provided"})
		return
	}

	// Exchange the code for a token and get user info
	userInfo, err := config.GetGoogleUserInfo(code, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info: " + err.Error()})
		return
	}

	// Find or create the user in our database
	user, err := h.oauthService.FindOrCreateUser("google", userInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find or create user: " + err.Error()})
		return
	}

	// Generate JWT token
	token, err := h.oauthService.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Create a session
	session = sessions.Default(c)
	session.Set("user_id", user.ID.Hex())
	session.Set("token", token)
	if err = session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
		return
	}

	// Redirect to the frontend with the token
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000" // Default to localhost:3000 if not set
	}
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, token)
	log.Printf("Redirecting to frontend: %s", redirectURL)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// Logout ends the user session and logs them out
func (h *OAuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{MaxAge: -1}) // Clear the session cookie
	err := session.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully logged out"})
}

// GetCurrentUser returns the currently authenticated user's ID
func (h *OAuthHandler) GetCurrentUser(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// In a real app, you would fetch the user from the database
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		// Add other user info as needed
	})
}
