package http_broadcast_nodes_updates

import (
	"encoding/json"
	"fmt"

	"github.com/horockey/service_discovery/internal/gateway/nodes_updates"
	"github.com/horockey/service_discovery/internal/gateway/nodes_updates/http_broadcast_nodes_updates/dto"
	"github.com/horockey/service_discovery/internal/model"
	"github.com/horockey/service_discovery/internal/repository/nodes"
)

var _ nodes_updates.Gateway = &httpBroadcastNodesUpdates{}

type httpBroadcastNodesUpdates struct {
	nodesRepo nodes.Repository
}

func (gw *httpBroadcastNodesUpdates) Send(upd model.Node) error {
	nodes, err := gw.nodesRepo.GetAll()
	if err != nil {
		return fmt.Errorf("getting list of nodes from repo: %w", err)
	}

	data, err := json.Marshal(dto.Node{
		ID:     upd.ID,
		Name:   upd.Name,
		IpAddr: upd.IpAddr.String(),
		State:  upd.State.String(),
	})
	if err != nil {
		return fmt.Errorf("marshaling data to json: %w", err)
	}

	for _, node := range nodes {
		if node.ID == upd.ID {
			continue
		}
		// TODO: send to workers chan
		_ = data // TODO: rm
	}
	return nil
}
