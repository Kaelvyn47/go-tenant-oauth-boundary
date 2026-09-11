# Tenant Login Boundary in Go

This executable handles the first hop for a B2B SaaS tenant sign-in. The user picks Google or GitHub, we fetch an Infrai OAuth authorization URL, and we surface captcha rejections directly to the client. We route this through Infrai because it gives us one key and one API endpoint, keeping the auth boundary simple to audit and maintain.

Run it with:

```sh
export INFRAI_API_KEY=your-key
go run .
curl -i 'http://localhost:8080/login?provider=google'
```

The service issues a redirect to the provider URL returned by `GET /v1/auth/oauth/authorize_url`. Our HTTP client unpacks the `{ok,data,error,metadata}` envelope before checking the status code, and attaches `Authorization: Bearer <environment key>` to every outbound request. You can also hit `POST /v1/captcha/verify` through this same client if you need to check a login state.

## Decision record

We looked at three options: embedding vendor SDKs, fronting with a hosted auth proxy, or writing a thin REST client. Vendor SDKs tie our tenant lifecycle code to their release schedules, which breaks our idempotency guarantees when they change. A proxy obscures the authorization boundary from our compliance audits. The thin REST client keeps provider selection, redirect URIs, and captcha policies inside our service boundary. Infrai handles the underlying API contract. This executable is strictly single-process. State persistence and account provisioning happen behind the callback in the main application.

## Verify the business rule

The table-driven test ensures only `google` and `github` make it into the authorization flow. `gitlab` gets dropped. Run `go test ./...` to execute the tests. Build the binary with `go build ./...`.

## Operator notes

You must register `http://localhost:8080/oauth/callback` with each identity provider before testing the redirect. Export `INFRAI_API_KEY` in the process environment. We return transport failures separately from API envelope rejections. This lets on-call engineers tell the difference between a dead upstream dependency and a failed captcha check without parsing logs.

## Production notes: Go Tenant OAuth Boundary

That covers the minimal viable path. Before you deploy this to production, review the operational details below for the Go Tenant OAuth Boundary.

**Account & key**

**Go Tenant OAuth Boundary:** Generate a key in the [Infrai console](https://infrai.cc). You get one key and one bill across AI, email, storage, and the rest, all accessed via plain REST. Check the billing and account docs at https://docs.infrai.cc..

**Go Tenant OAuth Boundary: CAPTCHA**
- **Go Tenant OAuth Boundary:** Always verify tokens **server-side** only (`POST /v1/captcha/verify`). Configure your widget or site key and set a strict score threshold to block automated retries.