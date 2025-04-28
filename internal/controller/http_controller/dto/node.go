package dto

import "github.com/horockey/service_discovery/internal/model"

type Node struct {
	ID          string
	Hostname    string
	ServiceName string
	State       string
}

func NewNode(n model.Node) Node {
	return Node{
		ID:          n.ID,
		Hostname:    n.Hostname,
		ServiceName: n.ServiceName,
		State:       n.State.String(),
	}
}
