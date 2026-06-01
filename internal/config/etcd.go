package config

import (
"context"
"encoding/json"
"log"
"strings"

clientv3 "go.etcd.io/etcd/client/v3"
)

const policyPrefix = "/vortex/policies/"

type Watcher struct {
client *clientv3.Client
store  *Store
}

func NewWatcher(client *clientv3.Client, store *Store) *Watcher {
return &Watcher{client: client, store: store}
}

func (w *Watcher) LoadInitial(ctx context.Context) error {
resp, err := w.client.Get(ctx, policyPrefix, clientv3.WithPrefix())
if err != nil {
return err
}
for _, kv := range resp.Kvs {
w.applyValue(kv.Key, kv.Value)
}
return nil
}

func (w *Watcher) Watch(ctx context.Context) {
ch := w.client.Watch(ctx, policyPrefix, clientv3.WithPrefix())
for watchResp := range ch {
if watchResp.Err() != nil {
log.Printf("etcd watch error: %v", watchResp.Err())
continue
}
for _, ev := range watchResp.Events {
switch ev.Type {
case clientv3.EventTypeDelete:
tenant := strings.TrimPrefix(string(ev.Kv.Key), policyPrefix)
w.store.Delete(tenant)
case clientv3.EventTypePut:
w.applyValue(ev.Kv.Key, ev.Kv.Value)
}
}
}
}

func (w *Watcher) applyValue(key []byte, raw []byte) {
var policy Policy
if err := json.Unmarshal(raw, &policy); err != nil {
log.Printf("invalid policy at key %s: %v", string(key), err)
return
}
if !policy.Valid() {
log.Printf("invalid policy contents at key %s", string(key))
return
}
w.store.Set(policy)
}
