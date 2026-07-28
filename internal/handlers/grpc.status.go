package handlers

import (
	"context"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
)

// NewGrpcHealthStatus converts an error into a DependencyHealth proto message,
// mapping nil to DEPENDENCY_STATUS_UP and any non-nil error to DEPENDENCY_STATUS_DOWN.
//
// The error is dropped from the message: raw dependency errors embed internal hostnames,
// ports and schema names. Operators read it from the trace span the failing probe records.
func NewGrpcHealthStatus(err error) *jsonkeysv2.DependencyHealth {
	return &jsonkeysv2.DependencyHealth{
		Status: lo.Ternary(
			err == nil,
			jsonkeysv2.DependencyStatus_DEPENDENCY_STATUS_UP,
			jsonkeysv2.DependencyStatus_DEPENDENCY_STATUS_DOWN,
		),
	}
}

// GrpcStatus is the gRPC handler that reports the operational health of the service
// and its dependencies.
type GrpcStatus struct {
	jsonkeysv2.UnimplementedStatusServiceServer
}

// NewGrpcStatus returns a new GrpcStatus handler.
func NewGrpcStatus() *GrpcStatus {
	return &GrpcStatus{}
}

func (handler *GrpcStatus) Status(
	ctx context.Context, _ *jsonkeysv2.StatusRequest,
) (*jsonkeysv2.StatusResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.Status")
	defer span.End()

	return otel.ReportSuccess(span, &jsonkeysv2.StatusResponse{
		Postgres: NewGrpcHealthStatus(handler.reportPostgres(ctx)),
	}), nil
}

func (handler *GrpcStatus) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "grpc.Status(reportPostgres)")
	defer span.End()

	pg, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		// Cannot assess the DB connection in transaction mode.
		return nil
	}

	err = pgdb.Ping()
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
