package health_upds

import (
	"context"

	"github.com/horockey/service_discovery/internal/model"
)

type Extractor interface {
	Start(context.Context) error
	Out() <-chan model.Node
}
