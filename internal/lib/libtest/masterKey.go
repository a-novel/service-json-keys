package libtest

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

// MasterKey is a fixed 32-byte master key.
var MasterKey = lo.Must(lib.ParseMasterKey("fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca"))
