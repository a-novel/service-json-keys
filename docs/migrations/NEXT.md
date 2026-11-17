# Upgrading to NEXT

The JavaScript client now builds on `RestClient` from `@a-novel-kit/nodelib-browser` 2.13.0, so it needs that version or a later one.

## Do I need to act?

**Required** only if you use `@a-novel/service-json-keys-rest` beside a `@a-novel-kit/nodelib-browser` older than 2.13.0. Every name and signature of the JavaScript client stays, and the Go client, the environment variables and the responses are unchanged, so every other consumer takes this release as-is.

## Install nodelib-browser 2.13 with the JavaScript client

`JsonKeysApi` now extends `RestClient`, which `@a-novel-kit/nodelib-browser` 2.13.0 introduced, so a consumer on an older nodelib-browser fails to build. Update nodelib-browser before the client, or in the same change:

```bash
pnpm add @a-novel-kit/nodelib-browser@^2.13.0
```

Its `fetch` and `fetchVoid` take a `RestInit`, a `RequestInit` that also accepts `accessToken` and `json`, and the client gains `fetchResponse` and `fetchBlob`. `fetchVoid` now releases the response body. `jwkList` and `jwkGet` send no `Content-Type`, since their requests carry no body.

## Notes

The REST ping opens no span, and the health check names its Postgres probe's span `rest.Health(client:postgres)` instead of `rest.Health(reportPostgres)`. Update a trace query or dashboard that filters on the old names.

No database migration or configuration change is required.
