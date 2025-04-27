package nodes

import (
	"context"

	"github.com/horockey/service_discovery/internal/model"
)

type Repository interface {
	GetAll(context.Context) ([]model.Node, error)
	Get(ctx context.Context, id string) (model.Node, error)
	AddOrUpdate(context.Context, model.Node) error
	Remove(ctx context.Context, id string) error
}
