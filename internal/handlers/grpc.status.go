package handlers

import (
	"context"

	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
)

// NewGrpcHealthStatus returns the health report for a successful dependency probe.
func NewGrpcHealthStatus() *jsonkeysv2.DependencyHealth {
	return &jsonkeysv2.DependencyHealth{
		Status: jsonkeysv2.DependencyStatus_DEPENDENCY_STATUS_UP,
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

// Status returns Unavailable when any dependency probe fails.
func (handler *GrpcStatus) Status(
	ctx context.Context, _ *jsonkeysv2.StatusRequest,
) (*jsonkeysv2.StatusResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.Status")
	defer span.End()

	err := handler.reportPostgres(ctx)
	if err != nil {
		_ = otel.ReportError(span, err)

		return nil, status.Error(codes.Unavailable, "service dependencies unavailable")
	}

	return otel.ReportSuccess(span, &jsonkeysv2.StatusResponse{
		Postgres: NewGrpcHealthStatus(),
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
		return otel.ReportError(span, postgres.ErrNoDbInContext)
	}

	err = pgdb.PingContext(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
