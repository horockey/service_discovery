package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/mux"
	"github.com/horockey/go-toolbox/http_helpers"
	controller_dto "github.com/horockey/service_discovery/internal/controller/http_controller/dto"
	"github.com/rs/zerolog"
)

const (
	healthEndpoint = "/health"
	updEndpoint    = "/updateMe"
)

type Node = controller_dto.Node

type Client struct {
	nodeID      string
	cl          *resty.Client
	serviceName string
	logger      zerolog.Logger
	out         chan Node

	serv *http.Server
}

func NewClient(
	serviceName string,
	baseURL string,
	apiKey string,
	serv *http.Server,
	logger zerolog.Logger,
) (*Client, error) {
	if serv == nil {
		return nil, errors.New("got nil serv")
	}
	return &Client{
		serviceName: serviceName,
		logger:      logger,
		cl: resty.New().
			SetBaseURL(baseURL).
			SetHeader("X-Api-Key", apiKey).
			SetRetryCount(3),
		serv: serv,
	}, nil
}

func (cl *Client) Register(
	ctx context.Context,
	hostname string,
	updCb func(Node) error,
) error {
	if updCb == nil {
		return errors.New("got nil callback")
	}

	router := mux.NewRouter()
	if cl.serv.Handler != nil {
		router.NotFoundHandler = cl.serv.Handler
	}

	router.HandleFunc(healthEndpoint, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)

	router.HandleFunc(updEndpoint, func(w http.ResponseWriter, req *http.Request) {
		n := controller_dto.Node{}
		if err := json.NewDecoder(req.Body).Decode(&n); err != nil {
			err = fmt.Errorf("decoding json: %w", err)
			cl.logger.Error().Err(err).Send()
			_ = http_helpers.RespondWithErr(w, http.StatusBadRequest, err)
			return
		}
		_ = req.Body.Close()

		if err := updCb(n); err != nil {
			cl.logger.
				Error().
				Err(fmt.Errorf("running upd callback: %w", err)).
				Send()
			_ = http_helpers.RespondWithErr(w, http.StatusInternalServerError, nil)
			return
		}
	}).Methods(http.MethodPost)

	cl.serv.Handler = router

	resp, err := cl.cl.R().
		SetContext(ctx).
		SetBody(controller_dto.RegisterNodeRequest{
			Hostname:       hostname,
			ServiceName:    cl.serviceName,
			HealthEndpoint: fmt.Sprintf("http://%s%s", hostname, healthEndpoint),
			UpdEndpoint:    fmt.Sprintf("http://%s%s", hostname, updEndpoint),
		}).
		Post("/node")
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("got non-ok response (%s): %s", resp.Status(), resp.String())
	}

	node := controller_dto.Node{}
	if err := json.Unmarshal(resp.Body(), &node); err != nil {
		return fmt.Errorf("unmarshaling json: %w", err)
	}

	cl.nodeID = node.ID

	return nil
}

func (cl *Client) Deregister(ctx context.Context) error {
	resp, err := cl.cl.R().
		SetContext(ctx).
		SetPathParam("nodeID", cl.nodeID).
		Delete("/node/{nodeID}")
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("got non-ok response (%s): %s", resp.Status(), resp.String())
	}

	return nil
}

func (cl *Client) GetNodes(ctx context.Context) ([]Node, error) {
	resp, err := cl.cl.R().
		SetContext(ctx).
		SetPathParam("serviceName", cl.serviceName).
		Get("/node/{serviceName}")
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("got non-ok response (%s): %s", resp.Status(), resp.String())
	}

	nodes := []controller_dto.Node{}
	if err := json.Unmarshal(resp.Body(), &nodes); err != nil {
		return nil, fmt.Errorf("unmarshaling json: %w", err)
	}

	return nodes, nil
}
