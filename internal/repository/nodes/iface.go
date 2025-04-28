package nodes

import (
	"context"

	"github.com/horockey/service_discovery/internal/model"
)

type Repository interface {
	// Get all nodes of given service. If serviceName set empty, rturns all nodes for all services
	GetAll(ctx context.Context) ([]model.Node, error)
	AddOrUpdate(context.Context, model.Node) error
	Remove(ctx context.Context, id string) error
}
