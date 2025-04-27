package nodes_updates

import (
	"context"

	"github.com/horockey/service_discovery/internal/model"
)

type Gateway interface {
	Send(ctx context.Context, upd model.Node) error
}
