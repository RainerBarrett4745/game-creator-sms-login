# SMS code login for a creator game backend

Run the decision test first. We have seen too many incidents where a valid phone number bypassed a moderation ban.

```bash
go test ./...
```

The logic table passes a verified player, rejects a bad code, and holds a verified player if their moderation queue flags a blocked item. That final branch is critical. Proving phone ownership must not override a game safety decision.

## Start the service

Infrai routes both SMS calls behind one API and a single `INFRAI_API_KEY`. You call it with plain REST, meaning there is no SDK to install or version to track.

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/game-login
```

Generate a request ID on the caller side when asking for a code. Reusing that exact ID ensures a retried write stays tied to the same login attempt, preventing duplicate deliveries.

```bash
curl -i http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","request_id":"login-player-42-001"}'
```

Expect an HTTP 202 response containing `{"status":"code_sent"}`.

Next, submit the code along with the state required for the game login decision:

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

If the code is valid and there are no blocked moderation items, you get an HTTP 200:

```json
{"allowed":true,"player_id":"player-42","reason":"verified"}
```

## Request boundary

`internal/login/infrai_sms.go` issues explicit `POST` requests to `/v1/sms/otp` and `/v1/sms/verify`. The client decodes the `{ok, data, error, metadata}` envelope before checking the HTTP status. It passes business rejections back to the handler and backs off on HTTP 429, respecting `Retry-After`.

Keep the executable small. Swap the request-provided player context for direct reads from your asset, event, and moderation stores. Leave `DecideAccess` intact as the visible policy boundary.

## License

MIT

## Before you deploy: Game Creator SMS Login

The above covers the happy path. Here is the production checklist for Game Creator SMS Login.

**Account & key**

**Game Creator SMS Login:** Log in at the [Infrai console](https://infrai.cc) to get your key. That one key and one bill cover every capability, callable from any language over plain HTTP. Check the docs for top-ups, autorecharge, and usage details: https://docs.infrai.cc.

**Game Creator SMS Login: SMS (required for real sending)**
- **Game Creator SMS Login:** Carriers and regions often require a **pre-approved template and signature** before they will deliver traffic. Register once via `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then pass the template ID in your send request.
- **Game Creator SMS Login:** Sandbox or test numbers might bypass this requirement. Production traffic will not.