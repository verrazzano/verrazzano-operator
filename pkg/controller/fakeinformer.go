// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

import (
	"time"
)

var _ cache.SharedIndexInformer = &FakeInformer{}

type FakeInformer struct {
	Synced   bool
	RunCount int
	handlers []cache.ResourceEventHandler
}

func (f *FakeInformer) LastSyncResourceVersion() string {
	return ""
}

func (f *FakeInformer) AddIndexers(indexers cache.Indexers) error {
	return nil
}

func (f *FakeInformer) GetIndexer() cache.Indexer {
	return nil
}

func (f *FakeInformer) Informer() cache.SharedIndexInformer {
	return f
}

func (f *FakeInformer) HasSynced() bool {
	return f.Synced
}

func (f *FakeInformer) AddEventHandler(handler cache.ResourceEventHandler) {
	f.handlers = append(f.handlers, handler)
}

func (f *FakeInformer) Run(<-chan struct{}) {
	f.RunCount++
}

func (f *FakeInformer) Add(obj metav1.Object) {
	for _, h := range f.handlers {
		h.OnAdd(obj)
	}
}

func (f *FakeInformer) Update(oldObj, newObj metav1.Object) {
	for _, h := range f.handlers {
		h.OnUpdate(oldObj, newObj)
	}
}

func (f *FakeInformer) Delete(obj metav1.Object) {
	for _, h := range f.handlers {
		h.OnDelete(obj)
	}
}

func (f *FakeInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) {

}

func (f *FakeInformer) GetStore() cache.Store {
	return nil
}

func (f *FakeInformer) GetController() cache.Controller {
	return nil
}

func (f *FakeInformer) SetWatchErrorHandler(handler cache.WatchErrorHandler) error {
	return nil
}
