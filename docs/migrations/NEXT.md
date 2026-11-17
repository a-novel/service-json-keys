# Upgrading to NEXT

The Go module no longer ships the `pkg/go/mocks` package. No repository imports it: a consumer mocks the narrow interface it owns.

## Do I need to act?

**Required** only if your code imports `github.com/a-novel/service-json-keys/v2/pkg/go/mocks`. The client, its request types and its token verifier are unchanged, so every other consumer takes this release as-is.

## Mock the interface your code owns

Declare the methods your package calls as its own interface, and generate its mock with your own mockery configuration.

```go
// before
import pkgmocks "github.com/a-novel/service-json-keys/v2/pkg/go/mocks"

client := pkgmocks.NewMockClient(t)

// after: an interface your package declares, mocked by your own mockery entry
type JsonKeysClient interface {
	JwkList(
		ctx context.Context, req *servicejsonkeys.JwkListRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.JwkListResponse, error)
}
```

`servicejsonkeys.Client` still satisfies any such interface, so production wiring stays the same.

## Notes

The gRPC server no longer opens a `core.JwkExportLocal` span: the signing key source reads its keys under the `core.JwkSearch` span directly. Update a trace query or dashboard that filters on the old name.

No database migration or configuration change is required.
