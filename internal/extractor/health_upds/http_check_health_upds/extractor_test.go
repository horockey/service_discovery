package http_check_health_upds_test

import (
	"context"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/horockey/service_discovery/internal/extractor/health_upds/http_check_health_upds"
	"github.com/horockey/service_discovery/internal/extractor/health_upds/http_check_health_upds/mock_nodes"
	"github.com/horockey/service_discovery/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const apiKey = "API_KEY"

var logger = zerolog.New(zerolog.ConsoleWriter{
	Out:        os.Stdout,
	TimeFormat: time.RFC3339,
}).With().Timestamp().Logger()

func TestExtractor(t *testing.T) {
	nodesRepo := mock_nodes.NewMockReadOnlyRepo(t)
	ex, err := http_check_health_upds.New(
		nodesRepo,
		10,
		time.Second,
		apiKey,
		logger,
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.TODO())

	var (
		node1Down = model.Node{
			ID:             "node1_id",
			Hostname:       "node1",
			ServiceName:    "fooBarService",
			State:          model.StateDown,
			HealthEndpoint: "http://localhost:9001/health",
		}
		node2Up = model.Node{
			ID:             "node2_id",
			Hostname:       "node2",
			ServiceName:    "fooBarService",
			State:          model.StateUp,
			HealthEndpoint: "http://localhost:9002/health",
		}
	)

	node1Up := *(&node1Down)
	node1Up.State = model.StateUp

	node2Down := *(&node2Up)
	node2Down.State = model.StateDown

	actualStates := []model.Node{node1Down, node2Up}

	nodesRepo.EXPECT().
		GetAll(ctx).
		Return(actualStates, nil)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	go http.ListenAndServe("localhost:9001", nil)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh := make(chan error, 1)
		t.Cleanup(func() {
			require.ErrorIs(t, <-errCh, context.Canceled)
		})
		errCh <- ex.Start(ctx)
		close(errCh)
	}()

	upds := []model.Node{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for upd := range ex.Out() {
			upds = append(upds, upd)
			idx := -1
			switch upd.Hostname {
			case "node1":
				idx = 0
			case "node2":
				idx = 1
			}
			newState := actualStates[idx]
			newState.State = upd.State
			actualStates[idx] = newState
		}
	}()

	time.Sleep(time.Second * 3)
	cancel()
	wg.Wait()

	assert.ElementsMatch(t, upds, []model.Node{node1Up, node2Down})
	assert.ElementsMatch(t, actualStates, []model.Node{node1Up, node2Down})
}
