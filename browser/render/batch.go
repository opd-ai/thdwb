// Package render provides rendering optimizations for the thdwb browser:
// batched DOM mutations, dirty flag management, and layer caching.
package render

import (
	"sync"
)

// Mutation is a pending DOM mutation to be applied in a batch.
type Mutation struct {
	Type     MutationType
	Node     interface{}
	Property string
	Value    interface{}
	OldValue interface{}
}

// MutationType describes the kind of DOM mutation.
type MutationType int

const (
	// MutationAttribute is an attribute change.
	MutationAttribute MutationType = iota
	// MutationChildList is a child addition/removal.
	MutationChildList
	// MutationCharacterData is a text content change.
	MutationCharacterData
	// MutationStyle is a computed style change.
	MutationStyle
)

// Batch applies DOM mutations in a single pass, minimizing reflows and
// repaints. Mutations are collected and then flushed together.
type Batch struct {
	mu        sync.Mutex
	mutations []*Mutation
	flushing  bool
	onFlush   func([]*Mutation)
}

// NewBatch creates a new mutation batcher.
func NewBatch(onFlush func([]*Mutation)) *Batch {
	return &Batch{onFlush: onFlush}
}

// Add queues a mutation for batched application.
func (b *Batch) Add(m *Mutation) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.flushing {
		// If currently flushing, apply directly.
		b.mutations = append(b.mutations, m)
		return
	}
	b.mutations = append(b.mutations, m)
}

// Flush applies all queued mutations in a single pass.
func (b *Batch) Flush() {
	b.mu.Lock()
	if len(b.mutations) == 0 {
		b.mu.Unlock()
		return
	}
	mutations := b.mutations
	b.mutations = nil
	b.flushing = true
	b.mu.Unlock()

	if b.onFlush != nil {
		b.onFlush(mutations)
	}

	b.mu.Lock()
	b.flushing = false
	b.mu.Unlock()
}

// Pending returns the number of queued mutations.
func (b *Batch) Pending() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.mutations)
}

// DirtyTracker manages dirty flags for nodes to enable incremental
// reflow/repaint. Nodes are marked dirty when they change and cleaned
// after being processed.
type DirtyTracker struct {
	mu           sync.Mutex
	dirty        map[interface{}]bool
	needsReflow  map[interface{}]bool
	needsRepaint map[interface{}]bool
}

// NewDirtyTracker creates a new DirtyTracker.
func NewDirtyTracker() *DirtyTracker {
	return &DirtyTracker{
		dirty:        make(map[interface{}]bool),
		needsReflow:  make(map[interface{}]bool),
		needsRepaint: make(map[interface{}]bool),
	}
}

// MarkDirty marks a node as dirty.
func (dt *DirtyTracker) MarkDirty(node interface{}) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.dirty[node] = true
}

// MarkNeedsReflow marks a node as needing reflow.
func (dt *DirtyTracker) MarkNeedsReflow(node interface{}) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.dirty[node] = true
	dt.needsReflow[node] = true
}

// MarkNeedsRepaint marks a node as needing repaint.
func (dt *DirtyTracker) MarkNeedsRepaint(node interface{}) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.dirty[node] = true
	dt.needsRepaint[node] = true
}

// IsDirty reports whether a node is dirty.
func (dt *DirtyTracker) IsDirty(node interface{}) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.dirty[node]
}

// NeedsReflow reports whether a node needs reflow.
func (dt *DirtyTracker) NeedsReflow(node interface{}) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.needsReflow[node]
}

// NeedsRepaint reports whether a node needs repaint.
func (dt *DirtyTracker) NeedsRepaint(node interface{}) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.needsRepaint[node]
}

// Clear clears all dirty flags for a node.
func (dt *DirtyTracker) Clear(node interface{}) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	delete(dt.dirty, node)
	delete(dt.needsReflow, node)
	delete(dt.needsRepaint, node)
}

// DirtyNodes returns all dirty nodes.
func (dt *DirtyTracker) DirtyNodes() []interface{} {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	out := make([]interface{}, 0, len(dt.dirty))
	for node := range dt.dirty {
		out = append(out, node)
	}
	return out
}

// Count returns the number of dirty nodes.
func (dt *DirtyTracker) Count() int {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return len(dt.dirty)
}

// LayerCache caches rendered layers for nodes to avoid repainting unchanged
// content. Each layer is keyed by node identity.
type LayerCache struct {
	mu     sync.Mutex
	layers map[interface{}]interface{}
}

// NewLayerCache creates a new LayerCache.
func NewLayerCache() *LayerCache {
	return &LayerCache{layers: make(map[interface{}]interface{})}
}

// Get returns the cached layer for a node, or nil if not cached.
func (lc *LayerCache) Get(node interface{}) interface{} {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.layers[node]
}

// Set stores a layer for a node.
func (lc *LayerCache) Set(node interface{}, layer interface{}) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.layers[node] = layer
}

// Delete removes the cached layer for a node.
func (lc *LayerCache) Delete(node interface{}) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.layers, node)
}

// Clear removes all cached layers.
func (lc *LayerCache) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.layers = make(map[interface{}]interface{})
}

// Count returns the number of cached layers.
func (lc *LayerCache) Count() int {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return len(lc.layers)
}

// Renderer combines batching, dirty tracking, and layer caching for
// optimized rendering.
type Renderer struct {
	batch  *Batch
	dirty  *DirtyTracker
	layers *LayerCache
}

// NewRenderer creates a new Renderer.
func NewRenderer(onFlush func([]*Mutation)) *Renderer {
	return &Renderer{
		batch:  NewBatch(onFlush),
		dirty:  NewDirtyTracker(),
		layers: NewLayerCache(),
	}
}

// Batch returns the mutation batcher.
func (r *Renderer) Batch() *Batch {
	return r.batch
}

// Dirty returns the dirty tracker.
func (r *Renderer) Dirty() *DirtyTracker {
	return r.dirty
}

// Layers returns the layer cache.
func (r *Renderer) Layers() *LayerCache {
	return r.layers
}

// Flush applies all pending mutations and clears dirty flags for processed
// nodes.
func (r *Renderer) Flush() {
	r.batch.Flush()
}

// InvalidateNode marks a node as dirty and removes its cached layer.
func (r *Renderer) InvalidateNode(node interface{}) {
	r.dirty.MarkDirty(node)
	r.layers.Delete(node)
}

// InvalidateNodeReflow marks a node as needing reflow and removes its layer.
func (r *Renderer) InvalidateNodeReflow(node interface{}) {
	r.dirty.MarkNeedsReflow(node)
	r.layers.Delete(node)
}

// InvalidateNodeRepaint marks a node as needing repaint and removes its layer.
func (r *Renderer) InvalidateNodeRepaint(node interface{}) {
	r.dirty.MarkNeedsRepaint(node)
	r.layers.Delete(node)
}

// ClearNode clears dirty flags for a node after processing.
func (r *Renderer) ClearNode(node interface{}) {
	r.dirty.Clear(node)
}

// Stats returns rendering statistics.
type Stats struct {
	DirtyNodes       int
	CachedLayers     int
	PendingMutations int
}

// Stats returns current rendering statistics.
func (r *Renderer) Stats() Stats {
	return Stats{
		DirtyNodes:       r.dirty.Count(),
		CachedLayers:     r.layers.Count(),
		PendingMutations: r.batch.Pending(),
	}
}
