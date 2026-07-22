package main

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

// recordingGenerator records the usages it was asked to rotate, and fails once it
// has been called failAfter times. Failing partway is the only case that
// distinguishes a real unit of work from writes that merely look grouped.
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

func twoUsages() map[string]*config.Jwk {
	return map[string]*config.Jwk{"auth": {}, "refresh": {}}
}

func TestRotateKeysProcessesEveryUsage(t *testing.T) {
	t.Parallel()

	generator := &recordingGenerator{}
	transactor := transactiontest.NewTransactor()

	processed, err := rotateKeys(t.Context(), transactor, generator, twoUsages())
	require.NoError(t, err)
	require.Equal(t, 2, processed)
	require.ElementsMatch(t, []string{"auth", "refresh"}, generator.usages)
	require.Equal(t, 1, transactor.Calls(), "every usage belongs to one unit of work, not one each")
}

// TestRotateKeysReportsNothingProcessedOnFailure covers the count's contract: a
// partial number describes work that has been rolled back, so reporting it would
// tell an operator that keys were rotated when none were.
func TestRotateKeysReportsNothingProcessedOnFailure(t *testing.T) {
	t.Parallel()

	generator := &recordingGenerator{failAfter: 1}

	processed, err := rotateKeys(t.Context(), transactiontest.NewTransactor(), generator, twoUsages())
	require.ErrorIs(t, err, errGenerate)
	require.Equal(t, 0, processed)
}

// TestRotateKeysIsOneUnitOfWork is the regression test for the defect this
// replaced: the rotation opened a transaction and then handed each generation the
// surrounding context, so every key committed on its own and a failure on a later
// usage left the earlier ones rotated.
//
// A transactor that refuses to open reproduces that boundary exactly. If the
// generations are inside the scope, none of them runs.
func TestRotateKeysIsOneUnitOfWork(t *testing.T) {
	t.Parallel()

	errNoTransaction := errors.New("transaction unavailable")

	generator := &recordingGenerator{}

	processed, err := rotateKeys(
		t.Context(), transactiontest.NewFailingTransactor(errNoTransaction), generator, twoUsages(),
	)
	require.ErrorIs(t, err, errNoTransaction)
	require.Zero(t, processed)
	require.Empty(t, generator.usages, "a generation ran outside the unit of work")
}
