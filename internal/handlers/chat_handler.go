// internal/handlers/chat_handler.go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// --- Structures pour la requête et la réponse Gemini ---

// GeminiRequestPart définit une partie d'un message.
type GeminiRequestPart struct {
	Text string `json:"text"`
}

// GeminiRequestContent définit un tour de conversation avec un rôle et des parties.
type GeminiRequestContent struct {
	Role  string              `json:"role"`
	Parts []GeminiRequestPart `json:"parts"`
}

// GeminiRequestBody est la structure principale du corps de la requête.
type GeminiRequestBody struct {
	Contents []GeminiRequestContent `json:"contents"`
}

// --- Structures pour parser la réponse de Gemini ---

// GeminiResponsePart définit une partie de la réponse.
type GeminiResponsePart struct {
	Text string `json:"text"`
}

// GeminiResponseCandidate contient une réponse potentielle du modèle.
type GeminiResponseCandidate struct {
	Content struct {
		Parts []GeminiResponsePart `json:"parts"`
		Role  string               `json:"role"`
	} `json:"content"`
}

// GeminiResponseBody est la structure principale de la réponse de l'API.
type GeminiResponseBody struct {
	Candidates []GeminiResponseCandidate `json:"candidates"`
}

// HandleChat gère les requêtes de chat en communiquant manuellement avec l'API Gemini.
func (h *Handler) HandleChat(c *gin.Context) {
	// 1. Lire le message de l'utilisateur depuis la requête entrante.
	// Nous attendons un format simple : {"message": "votre question ici"}
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format, expecting {\"message\": \"...\"}"})
		return
	}
	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message cannot be empty"})
		return
	}

	// 2. Construire l'URL et le corps de la requête pour l'API Gemini.
	apiKey := os.Getenv("GEMINI_API_KEY")
	// On utilise l'URL et le modèle que vous avez confirmés comme fonctionnels.
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey

	// Définition du "System Prompt" : les instructions et la personnalité du chatbot.
	systemPrompt := `You are a helpful and friendly assistant for the 'DentistFlow' dental clinic. You must follow these rules:
1. Your knowledge base is strictly limited to the following services and prices:
   - Standard Check-up: $75, Teeth Cleaning: $120, X-Ray: $50, Filling: $150-$300, Whitening: $400.
2. Answer questions politely based ONLY on this information.
3. If asked about anything else (e.g., opening hours, medical advice), you MUST respond with: "I can only provide information on our services and prices. For any other questions, please contact the clinic directly."
4. Do not make up services or prices.
5.You should be able to speak in all languages including Malagasy and automatically convert the price too accordin to the language of the user
6.If the user asks about the clinic's location, you must respond with: "Our clinic is located at 123 Main St, Anytown, USA. You can find us on Google Maps.
7.If the user asks about the clinic's opening hours, you must respond with: "Our clinic is open from 9:00 AM to 5:00 PM, Monday to Friday.apiKey.
8.If the user asks about the clinic's email, you must respond with: "Our clinic's email is rakotonarivomegane@gmail.com.`

	// Création du corps de la requête avec les instructions et la question de l'utilisateur.
	requestBody := GeminiRequestBody{
		Contents: []GeminiRequestContent{
			{
				Role:  "user", // Rôle "user" pour donner les instructions.
				Parts: []GeminiRequestPart{{Text: systemPrompt}},
			},
			{
				Role:  "model", // Rôle "model" pour simuler la confirmation des instructions.
				Parts: []GeminiRequestPart{{Text: "Understood. I will strictly follow these rules and only answer questions based on the provided service list."}},
			},
			{
				Role:  "user", // Rôle "user" pour la question réelle de l'utilisateur.
				Parts: []GeminiRequestPart{{Text: req.Message}},
			},
		},
	}

	// Conversion de la structure Go en JSON.
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request body"})
		return
	}

	// 3. Créer et envoyer la requête HTTP POST.
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create HTTP request"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send request to AI service"})
		return
	}
	defer httpResp.Body.Close()

	// 4. Lire et parser la réponse de Gemini.
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read AI response"})
		return
	}

	// Vérifier les codes d'erreur HTTP (ex: 400, 401, etc.).
	if httpResp.StatusCode != http.StatusOK {
		// Afficher l'erreur brute de Gemini dans la console du serveur pour le débogage.
		fmt.Printf("[DEBUG] Gemini Error Response: %s\n", string(respBody))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI service returned an error"})
		return
	}

	var geminiResp GeminiResponseBody
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	// 5. Extraire le message de la réponse et le renvoyer à notre frontend.
	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": geminiResp.Candidates[0].Content.Parts[0].Text,
		})
		return
	}

	// Message de secours si la réponse est vide ou mal formée.
	c.JSON(http.StatusInternalServerError, gin.H{"error": "AI returned an empty or invalid response"})
}
