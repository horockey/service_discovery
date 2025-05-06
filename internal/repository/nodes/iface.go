package nodes

import (
	"context"

	"github.com/horockey/service_discovery/internal/model"
)

type Repository interface {
	GetAll(ctx context.Context) ([]model.Node, error)
	AddOrUpdate(context.Context, model.Node) error
	SetDown(ctx context.Context, id string) error
}
