package watcher

import (
	"context"
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"git.henghajiang.com/backend/api_gateway_v2/core/utils"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"time"
)

type Watcher interface {
	Put(kv *mvccpb.KeyValue, isCreate bool) error
	Delete(kv *mvccpb.KeyValue) error
	BindTable(table *routing.Table)
	GetTable() *routing.Table
	GetWatchChan() clientv3.WatchChan
	Ctx() context.Context
	Refresh()
}

var Mapping map[Watcher]clientv3.WatchChan

func watch(w Watcher, c clientv3.WatchChan) {
	defer func() {
		if err := recover(); err != nil {
			stack := utils.Stack(3)
			logger.Errorf("[Recovery] %s panic recovered:\n%s\n%s", utils.TimeFormat(time.Now()), err, stack)
		}
		// restart watch func
		go watch(w, c)
	}()

	for {
		select {
		case <-w.Ctx().Done():
			logger.Exception(w.Ctx().Err())
			w.Refresh()
			c = w.GetWatchChan()
			Mapping[w] = c
			goto Over
		case resp := <-c:
			if resp.Canceled {
				logger.Warningf("watch canceled")
				logger.Exception(w.Ctx().Err())
				w.Refresh()
				c = w.GetWatchChan()
				Mapping[w] = c
				goto Over
			}
			if len(resp.Events) > 0 {
				for _, evt := range resp.Events {
					switch evt.Type {
					case mvccpb.PUT:
						if err := w.Put(evt.Kv, evt.IsCreate()); err != nil {
							logger.Exception(err)
						}
					case mvccpb.DELETE:
						if err := w.Delete(evt.Kv); err != nil {
							logger.Exception(err)
						}
					default:
						logger.Warningf("unrecognized event type: %d", evt.Type)
					}
				}
			}
		}
	}

Over:
	logger.Debugf("watch task finished")
}

func Watch(wch map[Watcher]clientv3.WatchChan) {
	for k, v := range wch {
		if k.GetTable() == nil {
			panic("watcher does not bind to routing table")
		}
		go watch(k, v)
	}
}