package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
)

var errGenerate = errors.New("generate")

// recordingGenerator records the usages it was asked to rotate, and fails once it has been
// called failAfter times.
type recordingGenerator struct {
	failAfter int
	usages    []string
}

func (generator *recordingGenerator) Exec(
	_ context.Context, request *core.JwkGenRequest,
) (*core.Jwk, error) {
	generator.usages = append(generator.usages, request.Usage)

	if generator.failAfter > 0 && len(generator.usages) >= generator.failAfter {
		return nil, errGenerate
	}

	return &core.Jwk{}, nil
}

func TestJwkRotateAll(t *testing.T) {
	t.Parallel()

	errNoTransaction := errors.New("transaction unavailable")

	testCases := []struct {
		name string

		failAfter  int
		transactor *transactiontest.Transactor

		expect       *core.JwkRotateAllResponse
		expectErr    error
		expectUsages []string
	}{
		{
			name: "Success",

			transactor: transactiontest.NewTransactor(),

			expect:       &core.JwkRotateAllResponse{Processed: 2},
			expectUsages: []string{"auth", "refresh"},
		},
		{
			// The second usage fails after the first succeeded. The count reports nothing, because
			// the transaction rolls the first one back.
			name: "Error/Generate",

			failAfter:  2,
			transactor: transactiontest.NewTransactor(),

			expectErr:    errGenerate,
			expectUsages: []string{"auth", "refresh"},
		},
		{
			// Every generation runs inside the unit of work, so none runs when it cannot open.
			name: "Error/Transaction",

			transactor: transactiontest.NewFailingTransactor(errNoTransaction),

			expectErr: errNoTransaction,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			generator := &recordingGenerator{failAfter: testCase.failAfter}
			keys := map[string]*config.Jwk{"auth": {}, "refresh": {}}

			service := core.NewJwkRotateAll(generator, testCase.transactor, keys)

			resp, err := service.Exec(t.Context(), &core.JwkRotateAllRequest{})
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)
			require.ElementsMatch(t, testCase.expectUsages, generator.usages)
			require.Equal(t, 1, testCase.transactor.Calls(), "every usage belongs to one unit of work")
		})
	}
}
