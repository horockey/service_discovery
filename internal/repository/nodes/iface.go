package nodes

import "github.com/horockey/service_discovery/internal/model"

type Repository interface {
	GetAll() ([]model.Node, error)
	Get(id string) (model.Node, error)
	AddOrUpdate(model.Node) error
	Remove(id string) error
}
