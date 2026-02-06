# Quick Start Guide - WACLI API

## Step 1: Build the API

```bash
cd /home/cslemes/projects/wacli
make build-api
```

Or manually:
```bash
go build -o bin/wacli-api cmd/wacli-api/main.go
```

## Step 2: Authenticate with WhatsApp (First Time Only)

The API uses the same authentication as the CLI. You need to authenticate once:

```bash
# Build CLI if not already built
make build-cli

# Show QR code to scan with WhatsApp mobile app
./bin/wacli auth qr
```

The session is stored in `~/.wacli/session.db` and will be used by the API.

## Step 3: Set API Keys

```bash
export WACLI_API_KEYS="your-secret-key-here"
```

You can use multiple keys separated by commas:
```bash
export WACLI_API_KEYS="key1,key2,key3"
```

## Step 4: Start the API Server

```bash
./bin/wacli-api
```

Or with custom settings:
```bash
export WACLI_API_PORT=3000
export GIN_MODE=release
./bin/wacli-api
```

The server will start on `http://0.0.0.0:8080` by default.

## Step 5: Test the API

### Health Check
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "service": "wacli-api"
}
```

### Check Authentication Status
```bash
curl http://localhost:8080/api/v1/auth/status \
  -H "X-API-Key: your-secret-key-here"
```

Expected response:
```json
{
  "authenticated": true,
  "connected": true
}
```

### Send a Test Message
```bash
curl -X POST http://localhost:8080/api/v1/send/text \
  -H "X-API-Key: your-secret-key-here" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "1234567890",
    "message": "Hello from WACLI API!"
  }'
```

### List Recent Messages
```bash
curl "http://localhost:8080/api/v1/messages?limit=10" \
  -H "X-API-Key: your-secret-key-here"
```

### Search Contacts
```bash
curl "http://localhost:8080/api/v1/contacts/search?q=john" \
  -H "X-API-Key: your-secret-key-here"
```

## All Available Endpoints

- `GET /health` - Health check (no auth required)
- `GET /api/v1/auth/status` - Check WhatsApp connection
- `GET /api/v1/messages` - List messages
- `GET /api/v1/messages/search?q=query` - Search messages
- `POST /api/v1/send/text` - Send text message
- `POST /api/v1/send/file` - Send file (multipart/form-data)
- `GET /api/v1/contacts` - List contacts
- `GET /api/v1/contacts/search?q=query` - Search contacts
- `GET /api/v1/chats` - List chats
- `GET /api/v1/groups` - List groups
- `POST /api/v1/sync` - Sync messages from WhatsApp

See [docs/api.md](docs/api.md) for complete API documentation.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `WACLI_API_KEYS` | Yes | - | Comma-separated API keys |
| `WACLI_API_HOST` | No | `0.0.0.0` | Host to bind to |
| `WACLI_API_PORT` | No | `8080` | Port to listen on |
| `WACLI_STORE_DIR` | No | `~/.wacli` | Data directory |
| `GIN_MODE` | No | `debug` | `debug` or `release` |

## Troubleshooting

### "not authenticated" error
Run `./bin/wacli auth qr` first to authenticate with WhatsApp.

### "connection failed" error
Check if WhatsApp session is valid:
```bash
./bin/wacli doctor
```

### Port already in use
Change the port:
```bash
export WACLI_API_PORT=3000
```

## Security Notes

⚠️ **Important Security Recommendations:**

1. **Use strong API keys** - Generate random, long keys
2. **Use HTTPS in production** - Deploy behind nginx/Caddy with TLS
3. **Keep keys secret** - Never commit to git
4. **Rotate keys regularly** - Change keys periodically
5. **Restrict access** - Use firewall rules or IP whitelisting

## Next Steps

- Read the [complete API documentation](docs/api.md)
- Check [deployment guide](docs/api-server.md)
- See [implementation details](docs/api-implementation.md)
- Try the [Python/JavaScript examples](docs/api.md#example-usage)

## Example Integration

### Python
```python
import requests

API_URL = 'http://localhost:8080'
API_KEY = 'your-secret-key-here'

def send_whatsapp(to, message):
    response = requests.post(
        f'{API_URL}/api/v1/send/text',
        headers={'X-API-Key': API_KEY},
        json={'to': to, 'message': message}
    )
    return response.json()

# Send message
result = send_whatsapp('1234567890', 'Hello!')
print(result)
```

### Node.js
```javascript
const axios = require('axios');

const API_URL = 'http://localhost:8080';
const API_KEY = 'your-secret-key-here';

async function sendWhatsApp(to, message) {
  const response = await axios.post(
    `${API_URL}/api/v1/send/text`,
    { to, message },
    { headers: { 'X-API-Key': API_KEY } }
  );
  return response.data;
}

// Send message
sendWhatsApp('1234567890', 'Hello!').then(console.log);
```

## Support

For issues or questions:
- Check the documentation in `docs/` folder
- Review the examples above
- Check existing GitHub issues
- Create a new issue if needed
