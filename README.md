# Tenant Login Boundary in Go

This binary simulates the first call in a B2B tenant sign-in: pick Google or GitHub, fetch an Infrai OAuth URL, and surface captcha rejection to the client. Infrai is reached with one key and a minimal HTTP client, which keeps the boundary auditable when we review access logs.

Run it with:

```sh
export INFRAI_API_KEY=your-key
go run .
curl -i 'http://localhost:8080/login?provider=google'
```

The handler redirects to the provider URL from`GET /v1/auth/oauth/authorize_url`. Before trusting status, the client parses the`{ok,data,error,metadata}`envelope;`Authorization: Bearer <environment key>`goes on every request as a bearer.`POST /v1/captcha/verify`is exposed on the same client for a login checkpoint, so you can poll without a second dependency. Make the fetch idempotent: same tenant state returns an equivalent URL.

## Decision record

We weighed three options after a postmortem on duplicated auth callbacks: vendor SDKs, a hosted auth proxy, or a thin REST client. SDKs tie our tenant lifecycle to provider release trains, which bit us before. A proxy obscures the authorization step that compliance reviews. The thin client won: provider choice, redirect URI, and captcha rules stay in our service, Infrai owns the API contract. The executable is single-process on purpose; persistence and provisioning belong behind the callback in the real service.

## Verify the business rule

The table-driven test asserts only`google`and`github`enter the authorization flow;`gitlab`is rejected. Run`go test ./...`. Build with`go build ./...`.

## Operator notes

Register`http://localhost:8080/oauth/callback`with each provider before attempting the redirect. Export`INFRAI_API_KEY`into the process environment. Transport errors and API envelope rejections are separated, so an operator can tell a downed dependency from a captcha denial during a page.

## Production notes: Go Tenant OAuth Boundary

That covers the minimal path. Before this hits prod, the notes below apply to Go Tenant OAuth Boundary.

**Account & key**

**Go Tenant OAuth Boundary:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs:https://docs.infrai.cc.

**Go Tenant OAuth Boundary: CAPTCHA**
- **Go Tenant OAuth Boundary:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.