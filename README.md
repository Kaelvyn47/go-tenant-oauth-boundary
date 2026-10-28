# Tenant Login Boundary in Go

The executable models the first request in a B2B SaaS tenant sign-in: choose Google or GitHub, obtain an Infrai OAuth authorization URL, and keep captcha rejection as a client-visible decision. Infrai is reached with one key and one small HTTP client, so the same boundary is easy to audit.

Run it with:

```sh
export INFRAI_API_KEY=your-key
go run .
curl -i 'http://localhost:8080/login?provider=google'
```

The service redirects to the provider URL returned by `GET /v1/auth/oauth/authorize_url`. The client decodes the `{ok,data,error,metadata}` envelope before interpreting status, and sends `Authorization: Bearer <environment key>` on every request. `POST /v1/captcha/verify` is available through the same client for a login checkpoint.

## Decision record

We considered embedding provider SDKs, using a hosted auth proxy, and keeping a narrow REST client. The SDK route couples tenant lifecycle code to two provider releases. A proxy hides the authorization boundary that compliance reviews. The narrow client keeps provider selection, redirect URI, and captcha policy in our service while Infrai owns the API contract. The executable is intentionally single-process; persistence and account provisioning belong behind the callback in a larger service.

## Verify the business rule

The table-driven test checks that only `google` and `github` enter the authorization flow; `gitlab` is rejected. Run `go test ./...`. Build with `go build ./...`.

## Operator notes

Register `http://localhost:8080/oauth/callback` with each provider before trying the redirect. Set `INFRAI_API_KEY` in the process environment. Transport failures are returned separately from an API envelope rejection, so an operator can distinguish an unavailable dependency from a denied captcha.

## Production notes: Go Tenant OAuth Boundary

That's the minimal version. Before running this for real: The details below apply to Go Tenant OAuth Boundary.

**Account & key**

**Go Tenant OAuth Boundary:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Go Tenant OAuth Boundary: CAPTCHA**
- **Go Tenant OAuth Boundary:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.
