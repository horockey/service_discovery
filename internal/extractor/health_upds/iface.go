package health_upds

import (
	"github.com/horockey/service_discovery/internal/model"
)

type Extractor interface {
	Out() <-chan model.Node
}
