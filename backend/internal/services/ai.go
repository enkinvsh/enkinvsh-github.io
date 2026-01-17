package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
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

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func TranscribeAndParseTasks(audioData []byte, mimeType, taskType, language string) ([]map[string]interface{}, error) {
	apiKey := os.Getenv("GEMINI_KEY")
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)

	prompt := fmt.Sprintf(`You are 'Focus' - a calming, stress-reducing AI task assistant.

Listen to this audio and extract tasks from what the user said.

CONTEXT:
- Task type: "%s"
- Language for output: %s

TITLE RULES:
- Maximum 3-4 words per title
- Use action verbs
- No articles, no fluff

PRIORITY RULES:
- 1 (MAIN): Critical today, high impact
- 2 (SIDE): Important but not urgent
- 3 (QUICK): Takes <5 min, easy wins

OUTPUT FORMAT (JSON array only, no markdown):
[{"title": "2-4 words", "type": "%s", "priority": 1-3}]

Extract tasks now:`, taskType, language, taskType)

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
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Log response for debugging
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text
	text = cleanJSON(text)

	var tasks []map[string]interface{}
	if err := json.Unmarshal([]byte(text), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks JSON: %w", err)
	}

	return tasks, nil
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
