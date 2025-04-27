package badger_nodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/horockey/service_discovery/internal/model"
	"github.com/horockey/service_discovery/internal/repository/nodes"
)

var _ nodes.Repository = &badgerNodes{}

type badgerNodes struct {
	db *badger.DB

	mu             sync.RWMutex
	downNodesRmDur time.Duration
	downedNodes    map[string]context.CancelFunc
}

func New(
	db *badger.DB,
	downNodesRmDur time.Duration,
) *badgerNodes {
	return &badgerNodes{
		db:             db,
		downNodesRmDur: downNodesRmDur,
		downedNodes:    map[string]context.CancelFunc{},
	}
}

func (repo *badgerNodes) GetAll(_ context.Context) ([]model.Node, error) {
	res := []model.Node{}

	err := repo.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			var data []byte
			_ = it.Item().Value(func(val []byte) error { data = val; return nil })

			n := model.Node{}
			if err := json.Unmarshal(data, &n); err != nil {
				return fmt.Errorf("unmarshalling data json: %w", err)
			}

			res = append(res, n)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("viewing db: %w", err)
	}

	return res, nil
}

func (repo *badgerNodes) Get(_ context.Context, id string) (model.Node, error) {
	n := model.Node{}
	err := repo.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(id))
		if err != nil {
			return fmt.Errorf("getting from db by key: %w", err)
		}

		data := []byte{}
		_ = item.Value(func(val []byte) error { data = val; return nil })

		if err := json.Unmarshal(data, &n); err != nil {
			return fmt.Errorf("unmarshalling data json: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.Node{}, fmt.Errorf("viewing db: %w", err)
	}

	return n, nil
}

func (repo *badgerNodes) AddOrUpdate(_ context.Context, n model.Node) error {
	err := repo.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(n)
		if err != nil {
			return fmt.Errorf("marshaling json: %w", err)
		}

		if err := txn.Set([]byte(n.ID), data); err != nil {
			return fmt.Errorf("setting kvp to bd: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("updating in db: %w", err)
	}

	switch n.State {
	case model.StateDown:
		ctx, cancel := context.WithTimeout(context.Background(), repo.downNodesRmDur)

		repo.mu.Lock()
		repo.downedNodes[n.ID] = cancel
		defer repo.mu.Unlock()

		go func(id string) {
			<-ctx.Done()
			switch {
			case errors.Is(ctx.Err(), context.Canceled):
				return
			case errors.Is(ctx.Err(), context.DeadlineExceeded):
				_ = repo.Remove(nil, id)
			}
		}(n.ID)
	case model.StateUp:
		repo.mu.Lock()
		cancel, found := repo.downedNodes[n.ID]
		if !found {
			repo.mu.Unlock()
			break
		}
		cancel()
		delete(repo.downedNodes, n.ID)
		repo.mu.Unlock()
	}

	return nil
}

func (repo *badgerNodes) Remove(_ context.Context, id string) error {
	err := repo.db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete([]byte(id)); err != nil {
			return fmt.Errorf("removing kvp: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("removing from bd: %w", err)
	}

	return nil
}
