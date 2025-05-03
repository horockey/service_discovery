package http_broadcast_nodes_updates

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/horockey/service_discovery/internal/gateway/nodes_updates"
	"github.com/horockey/service_discovery/internal/gateway/nodes_updates/http_broadcast_nodes_updates/dto"
	"github.com/horockey/service_discovery/internal/model"
	"github.com/rs/zerolog"
)

var _ nodes_updates.Gateway = &httpBroadcastNodesUpdates{}

var ErrClosed = errors.New("gateway is closed. Unable to write new message")

type httpBroadcastNodesUpdates struct {
	mu         sync.RWMutex
	closed     bool
	cl         *resty.Client
	sendCh     chan sendTask
	workersNum int
	logger     zerolog.Logger
}

type sendTask struct {
	msg      *model.Node
	endpoint string
}

func New(
	workersNum int,
	apiKey string,
	logger zerolog.Logger,
) (*httpBroadcastNodesUpdates, error) {
	if workersNum <= 0 {
		return nil, fmt.Errorf("workes num must be positive, got: %d", workersNum)
	}

	return &httpBroadcastNodesUpdates{
		workersNum: workersNum,
		logger:     logger,
		sendCh:     make(chan sendTask, workersNum),
		cl: resty.New().
			SetHeader("X-Api-Key", apiKey).
			SetHeader("Content-Type", "application/json").
			SetRetryCount(5),
	}, nil
}

func (gw *httpBroadcastNodesUpdates) Start(ctx context.Context) error {
	var wg sync.WaitGroup

	for range gw.workersNum {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range gw.sendCh {
				resp, err := gw.cl.R().
					SetContext(ctx).
					SetBody(dto.Node{
						ID:       task.msg.ID,
						Hostname: task.msg.Hostname,
						State:    task.msg.State.String(),
					}).
					Post(task.endpoint)
				if err != nil {
					gw.logger.
						Error().
						Str("endpoint", task.endpoint).
						Err(fmt.Errorf("executing request: %w", err)).
						Send()
					continue
				}
				if resp.StatusCode() != http.StatusOK {
					gw.logger.
						Error().
						Err(fmt.Errorf("got non-ok response (%s): %s", resp.Status(), resp.String())).
						Send()
					continue
				}
			}
		}()
	}

	<-ctx.Done()

	gw.mu.Lock()
	close(gw.sendCh)
	gw.closed = true
	gw.mu.Unlock()

	wg.Wait()

	return fmt.Errorf("running context: %w", ctx.Err())
}

func (gw *httpBroadcastNodesUpdates) Send(ctx context.Context, upd model.Node, recievers []model.Node) error {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	if gw.closed {
		return ErrClosed
	}

	for _, node := range recievers {
		if node.ID == upd.ID {
			continue
		}

		gw.sendCh <- sendTask{
			msg:      &upd,
			endpoint: node.UpdEndpoint,
		}
	}

	return nil
}
