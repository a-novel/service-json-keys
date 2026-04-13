package handlers

import (
	"context"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
)

// NewGrpcHealthStatus converts an error into a DependencyHealth proto message,
// mapping nil to DEPENDENCY_STATUS_UP and any non-nil error to DEPENDENCY_STATUS_DOWN.
func NewGrpcHealthStatus(err error) *protogen.DependencyHealth {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return &protogen.DependencyHealth{
		Status: lo.Ternary(
			err == nil,
			protogen.DependencyStatus_DEPENDENCY_STATUS_UP,
			protogen.DependencyStatus_DEPENDENCY_STATUS_DOWN,
		),
		Err: errMsg,
	}
}

// GrpcStatus is the gRPC handler that reports the operational health of the service
// and its dependencies.
type GrpcStatus struct {
	protogen.UnimplementedStatusServiceServer
}

// NewGrpcStatus returns a new GrpcStatus handler.
func NewGrpcStatus() *GrpcStatus {
	return new(GrpcStatus)
}

func (handler *GrpcStatus) Status(ctx context.Context, _ *protogen.StatusRequest) (*protogen.StatusResponse, error) {
	ctx, span := otel.Tracer().Start(ctx, "grpc.GrpcStatus")
	defer span.End()

	return &protogen.StatusResponse{
		Postgres: NewGrpcHealthStatus(handler.reportPostgres(ctx)),
	}, nil
}

func (handler *GrpcStatus) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "grpc.GrpcStatus(reportPostgres)")
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
