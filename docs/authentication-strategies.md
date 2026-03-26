# Authentication Strategies for HTTP Monitoring

Reference table explaining different authentication strategies for monitoring protected endpoints.

## When is authentication needed?

Most public websites (portfolios, company sites, blogs) don't require authentication to check if they're online — a simple GET request is enough. Authentication becomes necessary when monitoring **private or protected endpoints**.

## Authentication Types

| Auth Type | Use Case Example | How It Works | Config Complexity |
|---|---|---|---|
| **None** (default) | Public websites like `vsdpproductions.com`, `craft-agents.com` | Simple GET request, no credentials needed | None |
| **Basic Auth** | An admin panel at `https://admin.mysite.com` protected with username/password | Sends `Authorization: Basic <base64(user:pass)>` header with each request | Low |
| **API Key (header)** | A private API endpoint like `https://api.myservice.com/health` that requires `X-API-Key: abc123` | Sends a custom header with a static key | Low |
| **Bearer Token** | An OAuth-protected endpoint like `https://app.example.com/api/status` | Sends `Authorization: Bearer <token>` header | Medium (tokens expire) |
| **OAuth2 Client Credentials** | Machine-to-machine API access (e.g., cloud provider health endpoints) | Requests a token from an auth server, then uses Bearer Token | High |
| **Mutual TLS (mTLS)** | Strict zero-trust environments, internal microservices | Both client and server present certificates | High |

## Design Considerations

### Security of stored credentials
- **Never hardcode** credentials in source code
- Store in the YAML config file with restricted file permissions (`chmod 600`)
- For sensitive environments, consider environment variables or a secrets manager
- The config file should be in `.gitignore` to avoid committing credentials

### Token expiry
- Basic Auth and API Keys are static — no expiry management needed
- Bearer Tokens may expire — would need a refresh mechanism
- For a simple status checker, prefer static auth methods (Basic Auth, API Key)

### When NOT to use authentication in a status checker
- If the endpoint has IP-based allowlisting — no auth header needed
- If you only need to check "is the server responding?" — even a 401/403 means "the server is up"
- You can set `expected_status: 401` to monitor that a protected endpoint is reachable without needing credentials

## Example: Monitoring a protected endpoint without credentials
```yaml
  - name: "Admin Panel"
    url: "https://admin.mysite.com"
    expected_status: 401  # We expect 401 Unauthorized — that means the server IS up
```

This is a clever trick: you verify the server is responding without needing credentials.
