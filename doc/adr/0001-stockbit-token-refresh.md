# 0001: Stockbit token lifecycle with proactive and reactive refresh

Stockbit's API uses a short-lived access token paired with a rotating refresh token, and the server needs credentials to stay valid across restarts and idle periods. We decided to persist the token pair in Redis, drive it with both a proactive background refresh (ahead of expiry) and a reactive one (refresh-and-retry once on 401), serialize all refreshes so the rotating refresh token is never used concurrently, and fall back to a username/password login when the refresh token is rejected. Requests other than the login endpoint carry the access token automatically.

## Considered Options

- **In-memory tokens only** — simpler, but tokens were lost on every restart and forced an immediate login; Redis was already in the stack, so persistence was nearly free.
- **Proactive refresh only** — leaves a stale-token window after the background loop stops (e.g. long shutdown) where the first request 401s; reactive refresh closes that gap.
- **Reactive refresh only** — every idle restart pays a 401 before the first request succeeds; proactive refresh avoids it.
- **Unserialized refresh** — a concurrent refresh would use a token that rotation already invalidated, silently logging the account out.
