package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/steipete/wacli/internal/config"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Struct for Groq API response

type GroqTranscriptionResponse struct {
	Text string `json:"text"`
}

func TranscribeAudio(audioData []byte, apiKey string) (string, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the audio file directly from memory (no temp file needed)

	part, err := writer.CreateFormFile("file", "audio.ogg")

	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, bytes.NewReader(audioData)); err != nil {
		return "", fmt.Errorf("failed to copy audio content: %w", err)
	}

	// Model field

	if err := writer.WriteField("model", "whisper-large-v3"); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}

	writer.Close()

	// Build HTTP request

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/audio/transcriptions", body)

	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 20 * time.Second}

	resp, err := client.Do(req)

	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(errMsg))
	}

	var groqResp GroqTranscriptionResponse

	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return groqResp.Text, nil

}

func HandleMessages(ctx context.Context, client *whatsmeow.Client, evt interface{}, cfg *config.Config) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Message.GetAudioMessage() != nil {
			fmt.Println("🎙️ Received voice note from", v.Info.Sender.String())
			// Download audio
			audioData, err := client.Download(ctx, v.Message.GetAudioMessage())
			if err != nil {
				fmt.Println("Error downloading audio:", err)
				return
			}

			// Call Groq transcription

			transcript, err := TranscribeAudio(audioData, cfg.AI.GroqAPIKey)
			if err != nil {
				fmt.Println("❌ Transcription error:", err)
				return
			}

			// Build reply

			messageText := fmt.Sprintf("🎙️ *Transcrição do áudio:*\n\n\"%s\"\n\n_Powered by Cris AI 🤖_", transcript)
			quotedInfo := &waProto.ContextInfo{
				QuotedMessage: v.Message,
				Participant:   proto.String(v.Info.Sender.String()),
				StanzaID:      proto.String(v.Info.ID),
			}

			_, err = client.SendMessage(ctx, v.Info.Sender, &waProto.Message{
				ExtendedTextMessage: &waProto.ExtendedTextMessage{
					Text:        proto.String(messageText),
					ContextInfo: quotedInfo,
				},
			})

			if err != nil {
				fmt.Println("Error sending message:", err)
			}
		}
	}
}
