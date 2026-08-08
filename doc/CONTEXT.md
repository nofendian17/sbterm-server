# sbterm-server Context

This context covers the Stockbit Exodus API integration: how the server authenticates and keeps its Stockbit credentials valid.

## Language

**Access token**:
A short-lived bearer credential the Stockbit API accepts on protected endpoints. Attached as `Authorization: Bearer <access>` to every non-login request.
_Avoid_: token, session

**Refresh token**:
A longer-lived credential exchanged for a fresh access+refresh pair. It is what a refresh call authenticates with, never the access token.
_Avoid_: token, session

**Token rotation**:
Each refresh invalidates the previous refresh token, so a refresh token must never be used concurrently by two refreshes.
_Avoid_: reuse

**Login**:
Obtaining a fresh access+refresh pair from the configured username and password. Used when no pair exists yet or the refresh token was rejected.
_Avoid_: sign-in, re-auth

**Proactive refresh**:
Renewing the access token ahead of its expiry from a background loop, so requests never start from a stale token.
_Avoid_: timer, scheduler

**Reactive refresh**:
Renewing the access token after a request is rejected with 401, then retrying that request once.
_Avoid_: retry
