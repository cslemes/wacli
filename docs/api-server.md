# WACLI API Server

HTTP API for WhatsApp built with Gin framework and API key authentication.

## Quick Start

### 1. Build the API server

```bash
go build -o bin/wacli-api cmd/wacli-api/main.go
```

### 2. Set environment variables

```bash
export WACLI_API_KEYS="your-secret-key-1,your-secret-key-2"
export WACLI_API_PORT=8080  # optional, default: 8080
export WACLI_STORE_DIR="$HOME/.wacli"  # optional, default: ~/.wacli
export GIN_MODE="release"  # optional, default: debug
```

### 3. Run the server

```bash
./bin/wacli-api
```

Or use the Makefile:

```bash
make run-api
```

## Authentication

Before using the API, you need to authenticate with WhatsApp using the CLI:

```bash
# Build CLI first
make build-cli

# Authenticate with QR code
./bin/wacli auth qr

# Or use pairing code
./bin/wacli auth pair --phone 1234567890
```

The authentication session is stored in `$WACLI_STORE_DIR/session.db` and will be used by the API server.

## API Endpoints

See [api.md](api.md) for complete API documentation.

## Deployment

### Using Docker Compose

```bash
# Set API keys in .env file
echo "WACLI_API_KEYS=your-secret-key" > .env

# Start
docker-compose up -d

# Logs
docker-compose logs -f
```

### Using systemd

Create `/etc/systemd/system/wacli-api.service`:

```ini
[Unit]
Description=WACLI API Server
After=network.target

[Service]
Type=simple
User=wacli
WorkingDirectory=/opt/wacli
Environment="WACLI_API_KEYS=your-secret-key"
Environment="GIN_MODE=release"
Environment="WACLI_STORE_DIR=/var/lib/wacli"
ExecStart=/opt/wacli/bin/wacli-api
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable wacli-api
sudo systemctl start wacli-api
```

## Example Usage

### cURL

```bash
# Health check
curl http://localhost:8080/health

# Send message
curl -X POST http://localhost:8080/api/v1/send/text \
  -H "X-API-Key: your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"to": "1234567890", "message": "Hello!"}'

# List messages
curl http://localhost:8080/api/v1/messages?limit=10 \
  -H "X-API-Key: your-secret-key"

# Search contacts
curl "http://localhost:8080/api/v1/contacts/search?q=john" \
  -H "X-API-Key: your-secret-key"
```

### Python

```python
import requests

API_URL = 'http://localhost:8080'
API_KEY = 'your-secret-key'
headers = {'X-API-Key': API_KEY}

# Send message
response = requests.post(
    f'{API_URL}/api/v1/send/text',
    headers=headers,
    json={'to': '1234567890', 'message': 'Hello!'}
)
print(response.json())

# Upload and send file
files = {'file': open('photo.jpg', 'rb')}
data = {'to': '1234567890', 'caption': 'Check this out!'}
response = requests.post(
    f'{API_URL}/api/v1/send/file',
    headers=headers,
    files=files,
    data=data
)
print(response.json())
```

### JavaScript/Node.js

```javascript
const API_URL = 'http://localhost:8080';
const API_KEY = 'your-secret-key';

// Send text message
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

// Send file
async function sendFile(to, filePath, caption) {
  const formData = new FormData();
  formData.append('to', to);
  formData.append('caption', caption);
  formData.append('file', fs.createReadStream(filePath));

  const response = await fetch(`${API_URL}/api/v1/send/file`, {
    method: 'POST',
    headers: { 'X-API-Key': API_KEY },
    body: formData
  });
  return response.json();
}

// Usage
sendMessage('1234567890', 'Hello!').then(console.log);
```

## Security Best Practices

1. **Use Strong API Keys**: Generate cryptographically secure random keys
2. **Use HTTPS**: Deploy behind a reverse proxy (nginx, Caddy) with TLS
3. **Rate Limiting**: Implement rate limiting at the reverse proxy level
4. **Firewall**: Restrict access to trusted IPs only
5. **Environment Variables**: Never commit API keys to version control
6. **Rotate Keys**: Regularly rotate API keys

### Nginx Reverse Proxy Example

```nginx
server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Rate limiting
        limit_req zone=api burst=10 nodelay;
    }
}
```

## Troubleshooting

### "not authenticated" Error

Make sure you've authenticated using the CLI first:

```bash
./bin/wacli auth qr
```

The session is stored in `$WACLI_STORE_DIR/session.db`.

### Connection Issues

Check that WhatsApp session is still valid:

```bash
./bin/wacli doctor
```

### Port Already in Use

Change the port using the environment variable:

```bash
export WACLI_API_PORT=8081
./bin/wacli-api
```

## Performance Tips

1. **Use Release Mode**: Set `GIN_MODE=release` in production
2. **Connection Pooling**: The API reuses the same WhatsApp connection
3. **Database**: SQLite is used for local storage (fast for read operations)
4. **Caching**: Consider adding Redis for caching frequently accessed data

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

Returns:
```json
{
  "status": "ok",
  "service": "wacli-api"
}
```

### Auth Status

```bash
curl http://localhost:8080/api/v1/auth/status \
  -H "X-API-Key: your-secret-key"
```

Returns:
```json
{
  "authenticated": true,
  "connected": true
}
```

## API Limitations

- File uploads limited to 32 MB (Gin default)
- Long-running operations may timeout (adjust timeout in code if needed)
- Media download not yet fully implemented in API
- Some group management features require admin privileges
