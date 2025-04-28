package http_controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/horockey/go-toolbox/http_helpers"
	"github.com/horockey/service_discovery/internal/controller/http_controller/dto"
	"github.com/horockey/service_discovery/internal/model"
	"github.com/horockey/service_discovery/internal/usecase/discovery"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
)

type httpController struct {
	serv   *http.Server
	uc     *discovery.Usecase
	logger zerolog.Logger
	apiKey string
}

func New(
	addr string,
	uc *discovery.Usecase,
	logger zerolog.Logger,
) *httpController {
	ctrl := httpController{
		uc:     uc,
		logger: logger,
		serv:   &http.Server{Addr: addr},
	}

	router := mux.NewRouter()
	router.HandleFunc("/node", ctrl.handlePostNode).Methods(http.MethodPost)
	router.HandleFunc("/node", ctrl.handleGetNode).Methods(http.MethodGet)
	router.HandleFunc("/node/{serviceName}", ctrl.handleGetNodeServiceName).Methods(http.MethodGet)
	router.HandleFunc("/node/{nodeID}", ctrl.handleDeleteNodeId).Methods(http.MethodDelete)
	router.Use(ctrl.authMiddleware)

	ctrl.serv.Handler = router
	return &ctrl
}

func (ctrl *httpController) Start(ctx context.Context) (resErr error) {
	var wg sync.WaitGroup

	errCh := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- ctrl.serv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		sdCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if !errors.Is(ctx.Err(), context.Canceled) {
			resErr = errors.Join(resErr, fmt.Errorf("running context: %w", ctx.Err()))
		}
		if err := ctrl.serv.Shutdown(sdCtx); err != nil {
			resErr = errors.Join(resErr, fmt.Errorf("shutting down http server: %w", err))
		}

	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			resErr = errors.Join(resErr, fmt.Errorf("running http server: %w", err))
		}
	}

	return resErr
}

func (ctrl *httpController) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if ak := req.Header.Get("X-Api-Key"); ak != ctrl.apiKey {
			err := fmt.Errorf("bad api key %s", ak)
			ctrl.logger.Error().Err(err).Send()
			_ = http_helpers.RespondWithErr(w, http.StatusForbidden, err)
		}
	})
}

func (ctrl *httpController) handleDeleteNodeId(w http.ResponseWriter, req *http.Request) {
	nodeID, found := mux.Vars(req)["serviceName"]
	if !found {
		err := errors.New("missing nodeID")
		ctrl.logger.
			Error().
			Err(err).
			Send()
		_ = http_helpers.RespondWithErr(w, http.StatusBadRequest, err)
		return
	}

	if err := ctrl.uc.Deregister(req.Context(), nodeID); err != nil {
		ctrl.logger.
			Error().
			Err(fmt.Errorf("deregistering in usecase: %w", err)).
			Send()
		_ = http_helpers.RespondWithErr(w, http.StatusInternalServerError, nil)
		return
	}

	_ = http_helpers.RespondOK(w, nil)
}

func (ctrl *httpController) handlePostNode(w http.ResponseWriter, req *http.Request) {
	defer func() {
		_ = req.Body.Close()
	}()

	regNode := dto.RegisterNodeRequest{}
	if err := json.NewDecoder(req.Body).Decode(&regNode); err != nil {
		ctrl.logger.
			Error().
			Err(fmt.Errorf("decoding body json: %w", err)).
			Send()
		_ = http_helpers.RespondWithErr(w, http.StatusInternalServerError, nil)
		return
	}

	node, err := ctrl.uc.Register(req.Context(), model.RegisterNodeRequest(regNode))
	if err != nil {
		ctrl.logger.
			Error().
			Err(fmt.Errorf("registering in usecase: %w", err)).
			Send()
		_ = http_helpers.RespondWithErr(w, http.StatusInternalServerError, nil)
		return
	}

	_ = http_helpers.RespondOK(w, node)
}

func (ctrl *httpController) handleGetNode(w http.ResponseWriter, req *http.Request) {
	ctrl.getNodes("", w, req)
}

func (ctrl *httpController) handleGetNodeServiceName(w http.ResponseWriter, req *http.Request) {
	serviceName, found := mux.Vars(req)["serviceName"]
	if !found {
		err := errors.New("missing serviceName")
		ctrl.logger.
			Error().
			Err(err).
			Send()
		_ = http_helpers.RespondWithErr(w, http.StatusBadRequest, err)
		return
	}
	ctrl.getNodes(serviceName, w, req)
}

func (ctrl *httpController) getNodes(serviceName string, w http.ResponseWriter, req *http.Request) {
	nodes, err := ctrl.uc.GetAll(req.Context(), serviceName)
	if err != nil {
		ctrl.logger.
			Error().
			Err(fmt.Errorf("getting from usecase: %w", err)).
			Send()
		_ = http_helpers.RespondWithErr(w, http.StatusInternalServerError, nil)
		return
	}

	dtoNodes := lo.Map(
		nodes,
		func(el model.Node, _ int) dto.Node {
			return dto.NewNode(el)
		},
	)

	_ = http_helpers.RespondOK(w, dtoNodes)
}
