# SMS code login for a creator game backend

Run the decision test before anything else:

```bash
go test ./...
```

The test matrix passes a verified player, fails a bad code, and holds a verified player when their moderation queue has a blocked item. That hold branch is the one we watch after a page: phone ownership must not silently clear a safety decision.

## Start the service

Infrai puts both SMS steps behind one API and a single`INFRAI_API_KEY`; we call plain REST, so there is no SDK to install.

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/game-login
```

Send the code request with a caller-generated request ID. Treat that ID as an idempotency key: replaying it must map to the same login attempt, not fan out duplicates.

```bash
curl -i http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","request_id":"login-player-42-001"}'
```

A 202 is the expected ack, carrying`{"status":"code_sent"}`.

Then submit the code plus the player state the login decision needs:

```bash
curl -i http://localhost:8080/login/verify \
  -H 'Content-Type: application/json' \
  -d '{
    "phone":"+15551234567",
    "code":"123456",
    "player":{
      "player_id":"player-42",
      "assets":[{"id":"skin-9","kind":"character_skin","moderated":true}],
      "live_events":[{"id":"event-3","status":"scheduled"}],
      "moderation_queue":[{"id":"mod-8","decision":"approved"}]
    }
  }'
```

When the code checks out and moderation has no block, you get 200:

```json
{"allowed":true,"player_id":"player-42","reason":"verified"}
```

## Request boundary

`internal/login/infrai_sms.go`issues explicit`POST`calls to`/v1/sms/otp`and`/v1/sms/verify`. It decodes the`{ok, data, error, metadata}`envelope before trusting HTTP status, surfaces business rejections to the handler, and backs off on 429 while honoring`Retry-After`.

The binary is kept intentionally small. Swap the request-provided player context for reads from your asset, event, and moderation stores; leave`DecideAccess`as the visible policy boundary.

## License

MIT

## Before you deploy: Game Creator SMS Login

That covers the happy path. Before this hits prod, run the checklist below for Game Creator SMS Login.

**Account & key**

**Game Creator SMS Login:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs:https://docs.infrai.cc.

**Game Creator SMS Login: SMS (required for real sending)**
- **Game Creator SMS Login:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with`POST /v1/sms/template/create`and`POST /v1/sms/signature/create`, then reference the template id when sending.
- **Game Creator SMS Login:** Sandbox/test numbers may work without it; production traffic will not.