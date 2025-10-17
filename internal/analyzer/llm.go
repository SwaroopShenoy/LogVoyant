package analyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"logvoyant/internal/storage"
)

const groqAPIURL = "https://api.groq.com/openai/v1/chat/completions"

type GroqClient struct {
	apiKey string
	client *http.Client
}

type groqRequest struct {
	Model    string          `json:"model"`
	Messages []groqMessage   `json:"messages"`
	Temp     float64         `json:"temperature"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewGroqClient(apiKey string) *GroqClient {
	return &GroqClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (g *GroqClient) Analyze(prompt string) (*storage.Analysis, error) {
	reqBody := groqRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []groqMessage{
			{Role: "system", Content: "You are an expert log analyzer. Analyze logs and respond ONLY with valid JSON. No markdown, no code blocks, just pure JSON."},
			{Role: "user", Content: prompt},
		},
		Temp: 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", groqAPIURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("groq api error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var groqResp groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return nil, err
	}

	if len(groqResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from groq")
	}

	content := groqResp.Choices[0].Message.Content
	
	// Clean up markdown code blocks if present
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	
	var analysis storage.Analysis
	if err := json.Unmarshal([]byte(content), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analysis JSON: %w\nContent: %s", err, content)
	}

	return &analysis, nil
}