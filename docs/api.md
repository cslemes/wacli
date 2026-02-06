# WACLI HTTP API

REST API for WhatsApp CLI application using Gin framework with API key authentication.

## Setup

### Environment Variables

- `WACLI_API_KEYS` (required): Comma-separated list of valid API keys
- `WACLI_API_HOST` (optional): Host to bind to (default: "0.0.0.0")
- `WACLI_API_PORT` (optional): Port to listen on (default: 8080)
- `WACLI_STORE_DIR` (optional): Directory for WhatsApp session data (default: ~/.wacli)
- `GIN_MODE` (optional): "debug" or "release" (default: "debug")

### Running

```bash
# Install dependencies
go mod download

# Set API keys
export WACLI_API_KEYS="your-secret-key-1,your-secret-key-2"

# Run the API server
go run cmd/wacli-api/main.go
```

Or build and run:

```bash
go build -o wacli-api cmd/wacli-api/main.go
WACLI_API_KEYS="your-key" ./wacli-api
```

## Authentication

All endpoints require authentication using one of these methods:

1. **Header**: `X-API-Key: your-api-key`
2. **Query parameter**: `?api_key=your-api-key`
3. **Bearer token**: `Authorization: Bearer your-api-key`

## API Endpoints

### Health Check

```
GET /health
```

Returns service health status.

**Response:**
```json
{
  "status": "ok",
  "service": "wacli-api"
}
```

---

### Messages

#### List Messages

```
GET /api/v1/messages?chat=<jid>&limit=100&after=<RFC3339>&before=<RFC3339>
```

**Query Parameters:**
- `chat` (optional): Filter by chat JID
- `limit` (optional): Max results (default: 100)
- `after` (optional): RFC3339 timestamp
- `before` (optional): RFC3339 timestamp

**Response:**
```json
{
  "messages": [...],
  "fts": true
}
```

#### Search Messages

```
GET /api/v1/messages/search?q=<query>&chat=<jid>&limit=100
```

**Query Parameters:**
- `q` (required): Search query
- `chat` (optional): Filter by chat JID
- `limit` (optional): Max results (default: 100)

#### Get Message

```
GET /api/v1/messages/:id?chat=<jid>
```

**Query Parameters:**
- `chat` (required): Chat JID

---

### Sending Messages

#### Send Text Message

```
POST /api/v1/send/text
Content-Type: application/json

{
  "to": "1234567890",
  "message": "Hello from API"
}
```

**Response:**
```json
{
  "sent": true,
  "to": "1234567890@s.whatsapp.net",
  "id": "message-id"
}
```

#### Send File

```
POST /api/v1/send/file
Content-Type: multipart/form-data

to=1234567890
caption=Check this out
file=<binary file data>
```

**Response:**
```json
{
  "sent": true,
  "to": "1234567890@s.whatsapp.net",
  "id": "message-id",
  "filename": "photo.jpg"
}
```

---

### Contacts

#### List Contacts

```
GET /api/v1/contacts?limit=100
```

#### Search Contacts

```
GET /api/v1/contacts/search?q=<query>&limit=50
```

#### Get Contact

```
GET /api/v1/contacts/:jid
```

#### Set Contact Alias

```
POST /api/v1/contacts/:jid/alias
Content-Type: application/json

{
  "alias": "My Friend"
}
```

#### Refresh Contacts

```
POST /api/v1/contacts/refresh
```

Fetches latest contact information from WhatsApp.

---

### Chats

#### List Chats

```
GET /api/v1/chats?limit=100
```

#### Get Chat

```
GET /api/v1/chats/:jid
```

---

### Groups

#### List Groups

```
GET /api/v1/groups
```

#### Get Group Info

```
GET /api/v1/groups/:jid
```

#### Update Group Participants

```
POST /api/v1/groups/:jid/participants
Content-Type: application/json

{
  "action": "add",
  "participants": ["1234567890", "0987654321"]
}
```

**Actions:** `add`, `remove`, `promote`, `demote`

#### Update Group Name

```
POST /api/v1/groups/:jid/name
Content-Type: application/json

{
  "name": "New Group Name"
}
```

#### Get Group Invite Link

```
GET /api/v1/groups/:jid/invite?reset=false
```

**Query Parameters:**
- `reset` (optional): Reset the invite link (default: false)

#### Join Group

```
POST /api/v1/groups/join
Content-Type: application/json

{
  "invite_code": "abc123xyz"
}
```

#### Leave Group

```
POST /api/v1/groups/:jid/leave
```

---

### Authentication & Sync

#### Auth Status

```
GET /api/v1/auth/status
```

**Response:**
```json
{
  "authenticated": true,
  "connected": true
}
```

#### Sync Messages

```
POST /api/v1/sync
Content-Type: application/json

{
  "days": 30
}
```

Syncs message history from WhatsApp.

---

### Media

#### Download Media

```
GET /api/v1/media/:id?chat=<jid>
```

**Query Parameters:**
- `chat` (required): Chat JID

Returns the media file directly.

---

### History

#### Backfill History

```
POST /api/v1/history/backfill
Content-Type: application/json

{
  "chat_jid": "1234567890@s.whatsapp.net",
  "count": 100,
  "last_id": "message-id"
}
```

Request older messages for a specific chat.

---

## Example Usage

### Using curl

```bash
# Send a text message
curl -X POST http://localhost:8080/api/v1/send/text \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"to": "1234567890", "message": "Hello!"}'

# List messages
curl http://localhost:8080/api/v1/messages?limit=10 \
  -H "X-API-Key: your-api-key"

# Search contacts
curl http://localhost:8080/api/v1/contacts/search?q=john \
  -H "X-API-Key: your-api-key"
```

### Using JavaScript

```javascript
const API_URL = 'http://localhost:8080';
const API_KEY = 'your-api-key';

async function sendMessage(to, message) {
  const response = await fetch(`${API_URL}/api/v1/send/text`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY
    },
    body: JSON.stringify({ to, message })
  });
  return response.json();
}

// Send a message
sendMessage('1234567890', 'Hello from JavaScript!')
  .then(result => console.log('Sent:', result));
```

### Using Python

```python
import requests

API_URL = 'http://localhost:8080'
API_KEY = 'your-api-key'

headers = {'X-API-Key': API_KEY}

# Send text message
response = requests.post(
    f'{API_URL}/api/v1/send/text',
    headers=headers,
    json={'to': '1234567890', 'message': 'Hello from Python!'}
)
print(response.json())

# List messages
response = requests.get(
    f'{API_URL}/api/v1/messages',
    headers=headers,
    params={'limit': 10}
)
print(response.json())
```

## Notes

- The API requires an authenticated WhatsApp session (use the CLI to authenticate first: `wacli auth qr`)
- All timestamps use RFC3339 format
- JIDs (Jabber IDs) are WhatsApp's internal user identifiers (e.g., `1234567890@s.whatsapp.net`)
- Phone numbers should be in international format without '+' (e.g., `1234567890`)
- File uploads are limited by Gin's default settings (32 MB)
- Long-running operations (like sync) may timeout depending on your configuration
