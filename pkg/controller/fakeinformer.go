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

// FakeInformer impl
type FakeInformer struct {
	Synced   bool
	RunCount int
	handlers []cache.ResourceEventHandler
}

// LastSyncResourceVersion fake op
func (f *FakeInformer) LastSyncResourceVersion() string {
	return ""
}

// AddIndexers fake op
func (f *FakeInformer) AddIndexers(indexers cache.Indexers) error {
	return nil
}

// GetIndexer fake op
func (f *FakeInformer) GetIndexer() cache.Indexer {
	return nil
}

// Informer fake op
func (f *FakeInformer) Informer() cache.SharedIndexInformer {
	return f
}

// HasSynced fake op
func (f *FakeInformer) HasSynced() bool {
	return f.Synced
}

// AddEventHandler fake op
func (f *FakeInformer) AddEventHandler(handler cache.ResourceEventHandler) {
	f.handlers = append(f.handlers, handler)
}

// Run fake op
func (f *FakeInformer) Run(<-chan struct{}) {
	f.RunCount++
}

// Add fake op
func (f *FakeInformer) Add(obj metav1.Object) {
	for _, h := range f.handlers {
		h.OnAdd(obj)
	}
}

// Update fake op
func (f *FakeInformer) Update(oldObj, newObj metav1.Object) {
	for _, h := range f.handlers {
		h.OnUpdate(oldObj, newObj)
	}
}

// Delete fake op
func (f *FakeInformer) Delete(obj metav1.Object) {
	for _, h := range f.handlers {
		h.OnDelete(obj)
	}
}

// AddEventHandlerWithResyncPeriod fake op
func (f *FakeInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) {

}

// GetStore fake op
func (f *FakeInformer) GetStore() cache.Store {
	return nil
}

// GetController fake op
func (f *FakeInformer) GetController() cache.Controller {
	return nil
}

// SetWatchErrorHandler fake op
func (f *FakeInformer) SetWatchErrorHandler(handler cache.WatchErrorHandler) error {
	return nil
}
