package discovery

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/horockey/service_discovery/internal/extractor/health_upds"
	"github.com/horockey/service_discovery/internal/gateway/nodes_updates"
	"github.com/horockey/service_discovery/internal/model"
	"github.com/horockey/service_discovery/internal/repository/nodes"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
)

type Usecase struct {
	nodesRepo nodes.Repository
	upds      health_upds.Extractor
	gw        nodes_updates.Gateway
	logger    zerolog.Logger
}

func New(
	nodesRepo nodes.Repository,
	upds health_upds.Extractor,
	gw nodes_updates.Gateway,
	logger zerolog.Logger,
) *Usecase {
	return &Usecase{
		nodesRepo: nodesRepo,
		upds:      upds,
		gw:        gw,
		logger:    logger,
	}
}

func (uc *Usecase) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("running context: %w", ctx.Err())

		case upd := <-uc.upds.Out():
			if err := uc.nodesRepo.AddOrUpdate(ctx, upd); err != nil && !errors.Is(err, context.Canceled) {
				uc.logger.
					Error().
					Err(fmt.Errorf("updating repo: %w", err)).
					Send()
				continue
			}

			receivers, err := uc.nodesRepo.GetAll(ctx)
			if err != nil {
				uc.logger.
					Error().
					Err(fmt.Errorf("getting list of receivers from repo: %w", err)).
					Send()
				continue
			}
			receivers = lo.Filter(
				receivers,
				func(el model.Node, _ int) bool {
					return el.ServiceName == upd.ServiceName &&
						el.ID != upd.ID
				},
			)

			if err := uc.gw.Send(ctx, upd, receivers); err != nil {
				uc.logger.
					Error().
					Err(fmt.Errorf("sending upd to gw: %w", err)).
					Send()
				continue
			}
		}
	}
}

func (uc *Usecase) Register(ctx context.Context, req model.RegisterNodeRequest) (model.Node, error) {
	n := model.Node{
		ID:             uuid.NewString(),
		Hostname:       req.Hostname,
		ServiceName:    req.ServiceName,
		HealthEndpoint: req.HealthEndpoint,
		UpdEndpoint:    req.UpdEndpoint,
		State:          model.StateUp,
	}

	if err := uc.nodesRepo.AddOrUpdate(ctx, n); err != nil {
		return model.Node{}, fmt.Errorf("adding node to repo: %w", err)
	}

	return n, nil
}

func (uc *Usecase) Deregister(ctx context.Context, id string) error {
	if err := uc.nodesRepo.Remove(ctx, id); err != nil {
		return fmt.Errorf("removing node from repo: %w", err)
	}

	return nil
}

func (uc *Usecase) GetAll(ctx context.Context, serviceName string) ([]model.Node, error) {
	nodes, err := uc.nodesRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting node from")
	}

	return lo.Filter(
			nodes,
			func(el model.Node, _ int) bool {
				return el.ServiceName == serviceName
			},
		),
		nil
}
