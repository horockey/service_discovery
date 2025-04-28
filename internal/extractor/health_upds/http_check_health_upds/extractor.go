package http_check_health_upds

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/horockey/service_discovery/internal/extractor/health_upds"
	"github.com/horockey/service_discovery/internal/model"
	"github.com/rs/zerolog"
)

var _ health_upds.Extractor = &httpCheckHealthUpds{}

type httpCheckHealthUpds struct {
	nodesRepo     ReadOnlyRepo
	cl            *resty.Client
	out           chan model.Node
	checkInterval time.Duration
	logger        zerolog.Logger
}
type ReadOnlyRepo interface {
	GetAll(ctx context.Context) ([]model.Node, error)
}

func New(
	nodesRepo ReadOnlyRepo,
	outChSize int,
	checkInterval time.Duration,
	apiKey string,
	logger zerolog.Logger,
) (*httpCheckHealthUpds, error) {
	if outChSize <= 0 {
		return nil, fmt.Errorf("channel size num must be positive, got: %d", outChSize)
	}
	if checkInterval <= 0 {
		return nil, fmt.Errorf("check interval must be positive, got: %d", checkInterval)
	}

	return &httpCheckHealthUpds{
		nodesRepo:     nodesRepo,
		out:           make(chan model.Node, outChSize),
		checkInterval: checkInterval,
		logger:        logger,
		cl: resty.New().
			SetLogger(emptyRestyLogger{}).
			SetHeader("X-Api-Key", apiKey).
			SetRetryCount(5),
	}, nil
}

func (ex *httpCheckHealthUpds) Start(ctx context.Context) error {
	trig := make(chan time.Time, 1)
	trig <- time.Now()

	ticker := time.NewTicker(ex.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(ex.out)
			return fmt.Errorf("running context: %w", ctx.Err())
		case now := <-ticker.C:
			trig <- now
		case <-trig:
			upds, err := ex.getUpds(ctx)
			if err != nil {
				ex.logger.
					Error().
					Err(fmt.Errorf("getting updates: %w", err)).
					Send()
				continue
			}
			for _, upd := range upds {
				ex.out <- upd
			}
		}
	}
}

func (ex *httpCheckHealthUpds) Out() <-chan model.Node {
	return ex.out
}

func (ex *httpCheckHealthUpds) getUpds(ctx context.Context) ([]model.Node, error) {
	nodes, err := ex.nodesRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting nodes from repo: %w", err)
	}

	upds := []model.Node{}

	for _, node := range nodes {
		resp, err := ex.cl.R().
			SetContext(ctx).
			Get(node.HealthEndpoint)
		if err != nil {
			ex.logger.
				Error().
				Err(fmt.Errorf("executing request: %w", err)).
				Send()

			if node.State != model.StateDown {
				node.State = model.StateDown
				upds = append(upds, node)
			}

			continue
		}
		if resp.StatusCode() != http.StatusOK {
			ex.logger.
				Error().
				Err(fmt.Errorf("got non-ok response (%s): %s", resp.Status(), resp.String())).
				Send()

			if node.State != model.StateDown {
				node.State = model.StateDown
				upds = append(upds, node)
			}

			continue
		}

		if node.State != model.StateUp {
			node.State = model.StateUp
			upds = append(upds, node)
		}
	}

	return upds, nil
}
