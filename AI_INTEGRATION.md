# AI Integration - Voice Transcription

The AI module is now integrated with the wacli app using an event handler. It automatically transcribes voice messages and replies with the transcription.

## Setup

1. **Add your Groq API key to `.env`:**
   ```bash
   WACLI_AI_ENABLED=true
   GROQ_API_KEY=your-actual-groq-api-key-here
   ```

2. **Build the API server:**
   ```bash
   go build -tags sqlite_fts5 -o ./bin/wacli-api ./cmd/wacli-api
   ```

## How It Works

When the API server runs with AI enabled:

1. **Listens for incoming messages** during sync
2. **Detects voice/audio messages** automatically
3. **Downloads the audio** from WhatsApp
4. **Transcribes using Groq's Whisper API**
5. **Replies to the sender** with the transcription as a quoted message

## Running the API Server with AI

```bash
# Make sure .env has AI configuration
./bin/wacli-api
```

The AI handler will be active during any sync operations. When you receive a voice message, the server will:

```
🎙️ Received voice note from 1234567890@s.whatsapp.net
[Downloading and transcribing...]
[Sending reply with transcription]
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `WACLI_AI_ENABLED` | Enable/disable AI features | `false` |
| `GROQ_API_KEY` | Your Groq API key for transcription | (required) |

## Architecture

The integration follows this flow:

```
Message Event
    ↓
Event Handler (app/sync.go)
    ↓
AI Handler (ai/ai.go)
    ↓
1. Check if it's an audio message
2. Download audio via WhatsApp client
3. Send to Groq API for transcription
4. Send reply back to sender
```

## Testing

1. Start the API server with AI enabled
2. Authenticate with WhatsApp (if not already)
3. Send yourself a voice message from your phone
4. The server will automatically transcribe and reply

## API Response Format

The bot replies with:

```
🎙️ *Transcrição do áudio:*

"[Transcribed text here]"

_Powered by Cris AI 🤖_
```

The reply is sent as a quoted message, referencing the original voice note.

## Notes

- Only works during active sync (when the server is connected to WhatsApp)
- Transcription uses Groq's `whisper-large-v3` model
- Audio is processed in memory (no temp files)
- Replies are sent to the same chat where the voice message was received
