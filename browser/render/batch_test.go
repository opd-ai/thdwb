package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchAddAndFlush(t *testing.T) {
	var applied []*Mutation
	b := NewBatch(func(ms []*Mutation) {
		applied = ms
	})

	b.Add(&Mutation{Type: MutationAttribute, Node: "node1", Property: "class", Value: "foo"})
	b.Add(&Mutation{Type: MutationChildList, Node: "node2", Property: "children", Value: "add"})

	require.Equal(t, 2, b.Pending())
	b.Flush()
	require.Len(t, applied, 2)
	assert.Equal(t, MutationAttribute, applied[0].Type)
	assert.Equal(t, "node1", applied[0].Node)
	assert.Equal(t, "class", applied[0].Property)
	assert.Equal(t, MutationChildList, applied[1].Type)
	assert.Equal(t, 0, b.Pending())
}

func TestBatchFlushEmpty(t *testing.T) {
	var applied []*Mutation
	b := NewBatch(func(ms []*Mutation) {
		applied = ms
	})

	b.Flush() // no-op
	assert.Nil(t, applied)
	assert.Equal(t, 0, b.Pending())
}

func TestBatchAddDuringFlush(t *testing.T) {
	var applied []*Mutation
	var b *Batch
	b = NewBatch(func(ms []*Mutation) {
		applied = append(applied, ms...)
	})

	b.Add(&Mutation{Type: MutationAttribute, Node: "node1", Property: "id", Value: "a"})
	b.Add(&Mutation{Type: MutationStyle, Node: "node2", Property: "color", Value: "red"})
	b.Flush()

	require.Len(t, applied, 2)
	assert.Equal(t, "node1", applied[0].Node)
	assert.Equal(t, "node2", applied[1].Node)
	assert.Equal(t, 0, b.Pending())
}

func TestDirtyTracker(t *testing.T) {
	dt := NewDirtyTracker()

	node1 := "node1"
	node2 := "node2"

	dt.MarkDirty(node1)
	assert.True(t, dt.IsDirty(node1))
	assert.False(t, dt.IsDirty(node2))

	dt.MarkNeedsReflow(node1)
	assert.True(t, dt.NeedsReflow(node1))

	dt.MarkNeedsRepaint(node2)
	assert.True(t, dt.NeedsRepaint(node2))

	dt.Clear(node1)
	assert.False(t, dt.IsDirty(node1))
	assert.False(t, dt.NeedsReflow(node1))
	assert.False(t, dt.NeedsRepaint(node1))
	assert.True(t, dt.NeedsRepaint(node2)) // unaffected
}

func TestDirtyTrackerDirtyNodes(t *testing.T) {
	dt := NewDirtyTracker()

	dt.MarkDirty("a")
	dt.MarkDirty("b")
	dt.MarkDirty("c")

	nodes := dt.DirtyNodes()
	require.Len(t, nodes, 3)
	assert.Equal(t, 3, dt.Count())
}

func TestLayerCache(t *testing.T) {
	lc := NewLayerCache()

	assert.Nil(t, lc.Get("node1"))

	lc.Set("node1", "layer1")
	assert.Equal(t, "layer1", lc.Get("node1"))

	lc.Delete("node1")
	assert.Nil(t, lc.Get("node1"))

	lc.Set("node2", "layer2")
	assert.Equal(t, 1, lc.Count())

	lc.Clear()
	assert.Equal(t, 0, lc.Count())
}

func TestRenderer(t *testing.T) {
	var applied []*Mutation
	r := NewRenderer(func(ms []*Mutation) {
		applied = ms
	})

	require.NotNil(t, r.Batch())
	require.NotNil(t, r.Dirty())
	require.NotNil(t, r.Layers())

	node := "node1"
	r.InvalidateNode(node)
	assert.True(t, r.Dirty().IsDirty(node))
	assert.Nil(t, r.Layers().Get(node))

	r.Batch().Add(&Mutation{Type: MutationAttribute, Node: node, Property: "class", Value: "x"})
	assert.Equal(t, 1, r.Batch().Pending())

	r.Flush()
	require.Len(t, applied, 1)
	assert.Equal(t, 0, r.Batch().Pending())

	stats := r.Stats()
	assert.Equal(t, 1, stats.DirtyNodes)
	assert.Equal(t, 0, stats.PendingMutations)
}

func TestRendererInvalidateReflowAndRepaint(t *testing.T) {
	r := NewRenderer(nil)

	node := "node1"
	r.InvalidateNodeReflow(node)
	assert.True(t, r.Dirty().IsDirty(node))
	assert.True(t, r.Dirty().NeedsReflow(node))

	r.InvalidateNodeRepaint(node)
	assert.True(t, r.Dirty().NeedsRepaint(node))

	r.ClearNode(node)
	assert.False(t, r.Dirty().IsDirty(node))
	assert.False(t, r.Dirty().NeedsReflow(node))
	assert.False(t, r.Dirty().NeedsRepaint(node))
}
