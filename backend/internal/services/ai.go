package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []GeminiSafetySetting   `json:"safetySettings,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text       string        `json:"text,omitempty"`
	InlineData *GeminiInline `json:"inline_data,omitempty"`
}

type GeminiInline struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type GeminiGenerationConfig struct {
	Temperature      float64 `json:"temperature,omitempty"`
	ResponseMIMEType string  `json:"responseMimeType,omitempty"`
}

type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GeminiResponse struct {
	Candidates     []GeminiCandidate     `json:"candidates"`
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
}

type GeminiCandidate struct {
	Content       GeminiContent `json:"content"`
	FinishReason  string        `json:"finishReason,omitempty"`
	SafetyRatings []struct {
		Category    string `json:"category"`
		Probability string `json:"probability"`
	} `json:"safetyRatings,omitempty"`
}

type GeminiPromptFeedback struct {
	BlockReason   string `json:"blockReason,omitempty"`
	SafetyRatings []struct {
		Category    string `json:"category"`
		Probability string `json:"probability"`
	} `json:"safetyRatings,omitempty"`
}

type Task struct {
	Title    string `json:"title"`
	Type     string `json:"type"`
	Priority int    `json:"priority"`
}

func TranscribeAndParseTasks(audioData []byte, mimeType, taskType, language string) ([]map[string]interface{}, error) {
	proxyURL := os.Getenv("GEMINI_PROXY_URL")
	if proxyURL == "" {
		proxyURL = "https://focus.enkinvsh.workers.dev"
	}
	url := fmt.Sprintf("%s?model=gemini-2.0-flash", proxyURL)

	prompt := fmt.Sprintf(`Listen to this audio and extract tasks from what the user said.

Task type: "%s"
Language for output: %s

Create tasks with these rules:
- Title: 2-4 words, action verb, no articles
- Type: "%s"
- Priority: 1 (urgent), 2 (important), or 3 (quick)

Return a JSON array of tasks only.`, taskType, language, taskType)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{{
			Parts: []GeminiPart{
				{InlineData: &GeminiInline{
					MimeType: mimeType,
					Data:     base64.StdEncoding.EncodeToString(audioData),
				}},
				{Text: prompt},
			},
		}},
		GenerationConfig: &GeminiGenerationConfig{
			Temperature:      0.1,
			ResponseMIMEType: "application/json",
		},
		SafetySettings: []GeminiSafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_NONE"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_NONE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_NONE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_NONE"},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Printf("Gemini request size: %d bytes, mimeType: %s", len(audioData), mimeType)

	// Use HTTP client with 30s timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("Gemini response (status %d): %s", resp.StatusCode, string(body))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response JSON: %w, body: %s", err, string(body))
	}

	if geminiResp.PromptFeedback != nil && geminiResp.PromptFeedback.BlockReason != "" {
		return nil, fmt.Errorf("prompt blocked: %s", geminiResp.PromptFeedback.BlockReason)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned from Gemini: %s", string(body))
	}

	candidate := geminiResp.Candidates[0]

	if candidate.FinishReason == "SAFETY" {
		return nil, fmt.Errorf("response blocked by safety filters")
	}

	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in response: %s", string(body))
	}

	text := candidate.Content.Parts[0].Text
	if text == "" {
		return nil, fmt.Errorf("empty text in response: %s", string(body))
	}

	log.Printf("Gemini extracted text: %s", text)

	var tasks []Task
	if err := json.Unmarshal([]byte(text), &tasks); err != nil {
		cleaned := cleanJSON(text)
		if err := json.Unmarshal([]byte(cleaned), &tasks); err != nil {
			return nil, fmt.Errorf("failed to parse tasks JSON: %w, text: %s", err, text)
		}
	}

	var result []map[string]interface{}
	for _, task := range tasks {
		result = append(result, map[string]interface{}{
			"title":    task.Title,
			"type":     task.Type,
			"priority": task.Priority,
		})
	}

	return result, nil
}

func cleanJSON(s string) string {
	start := -1
	end := -1
	for i, c := range s {
		if c == '[' && start == -1 {
			start = i
		}
		if c == ']' {
			end = i + 1
		}
	}
	if start != -1 && end != -1 && end > start {
		return s[start:end]
	}
	return s
}
