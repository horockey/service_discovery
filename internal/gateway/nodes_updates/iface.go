package nodes_updates

import "github.com/horockey/service_discovery/internal/model"

type Gateway interface {
	Send(upd model.Node) error
}
