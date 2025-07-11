---
outline: deep
---

# Go module

You can interact with a running service by using the exported Package.

```bash
go get github.com/a-novel/service-json-keys
```

## Setup

You need an API client to interact with the running service. Make sure to provide the correct URL:

```go
package main

import jkPkg "github.com/a-novel/service-json-keys/pkg"

// serverURL := "http://localhost:4001/v1"
client, err := jkPkg.NewAPIClient(serverURL)
```

## Sign / Verify claims

You can produce and consume JWTs. To do so, you need to select a config. Available configs are listed using
the `models.KeyUsage` constants.

```go
package main

import jkModels "github.com/a-novel/service-json-keys/models"

// Tokens signed with a given usage are only valid for that usage.
usage := jkModels.KeyUsageAuth
```

Sign a claim

```go
package main

import jkPkg "github.com/a-novel/service-json-keys/pkg"

type CustomClaims struct {
	UserID string `json:"userID"`
}

// You can share a single signer for all usages.
signer := jkPkg.NewClaimsSigner(client)

token, err := signer.SignClaims(ctx, usage, &CustomClaims{)
	UserID: "1234",
})
```

Consume claims

```go
package main

import jkPkg "github.com/a-novel/service-json-keys/pkg"

type CustomClaims struct {
	UserID string `json:"userID"`
}

// Each claim verifier is only targeted at a specific claims body type.
// You may share a single verifier for multiple usages, if the token
// body type is the same.
verifier, err := jkPkg.NewClaimsVerifier[CustomClaims](client)

// Note the token is also validated, meaning expired tokens will return an error.
claims, err := signer.SignClaims(ctx, usage, token, nil)
```
