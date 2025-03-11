package notifier

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-btc/common/tasks"
	"github.com/dapplink-labs/multichain-sync-btc/database"
)

type Notifier struct {
	db             *database.DB
	businessIds    []string
	notifyClient   map[string]*NotifyClient
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker

	shutdown context.CancelCauseFunc
	stopped  atomic.Bool
}

func NewNotifier(db *database.DB, shutdown context.CancelCauseFunc) (*Notifier, error) {
	businessList, err := db.Business.QueryBusinessList()
	if err != nil {
		log.Error("query business list fail", "err", err)
		return nil, err
	}

	var businessIds []string
	var notifyClient map[string]*NotifyClient
	for _, business := range businessList {
		log.Info("handle business id", "business", business.BusinessUid)
		//businessIds = append(businessIds, business.BusinessUid)
		//client, err := NewNotifierClient(business.NotifyUrl)
		//if err != nil {
		//	log.Error("new notify client fail", "err", err)
		//	return nil, err
		//}
		//notifyClient[business.BusinessUid] = client
	}

	resCtx, resCancel := context.WithCancel(context.Background())
	return &Notifier{
		db:             db,
		notifyClient:   notifyClient,
		businessIds:    businessIds,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in internals: %w", err))
		}},
		ticker: time.NewTicker(time.Second * 5),
	}, nil
}

func (nf *Notifier) Start(ctx context.Context) error {
	log.Info("start notifier......")
	nf.tasks.Go(func() error {
		for {
			select {
			case <-nf.ticker.C:
				log.Info("start notifier")
			case <-nf.resourceCtx.Done():
				log.Info("stop notifier")
				return nil
			}
		}
	})
	return nil
}

func (nf *Notifier) Stop(ctx context.Context) error {
	var result error
	nf.resourceCancel()
	nf.ticker.Stop()
	if err := nf.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await notify %w"), err)
		return result
	}
	log.Info("stop notify success")
	return nil
}

func (nf *Notifier) Stopped() bool {
	return nf.stopped.Load()
}
