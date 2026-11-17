# Migration guides

- [v2.16.0](./v2.16.0.md): The Go module no longer ships the `pkg/go/mocks` package. No repository in the a-novel or a-novel-kit organizations imports it: a consumer mocks the narrow interface it owns.
- [v2.13.0](v2.13.0.md): Configure PostgreSQL through discrete settings, not a URL.
- [v2.11.0](v2.11.0.md): Handle an unknown `ClaimsSign` usage as `INVALID_ARGUMENT`.
