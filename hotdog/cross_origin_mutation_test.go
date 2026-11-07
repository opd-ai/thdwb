package hotdog

import (
	"testing"

	"github.com/danfragoso/thdwb/assets"
	profiler "github.com/danfragoso/thdwb/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckCrossOriginMutationRejects (audit R-8) verifies the Go-level DOM
// mutation guard actually rejects cross-origin appends. The function used to
// fall through to `return nil` in the cross-origin branch, so the guard in
// appendChildNode never blocked anything -- defense-in-depth against iframe
// document contamination was silently dead.
func TestCheckCrossOriginMutationRejects(t *testing.T) {
	a := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer a.Destroy()
	b := NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())
	defer b.Destroy()
	require.NoError(t, a.SetOrigin("https://top.example"))
	require.NoError(t, b.SetOrigin("https://iframe.example"))

	parent := &NodeDOM{Type: NodeTypeElement, Element: "div", WindowCtx: a, Children: []*NodeDOM{}}
	child := &NodeDOM{Type: NodeTypeElement, Element: "span", WindowCtx: b}

	err := parent.checkCrossOriginMutation(child)
	require.Error(t, err, "cross-origin append must be rejected")
	assert.Contains(t, err.Error(), "cross-origin")

	parent.appendChildNode(child)
	assert.Empty(t, parent.Children, "appendChildNode must not link a cross-origin child")
	assert.Nil(t, child.Parent)

	// Same-origin window contexts are still allowed.
	sibling := &NodeDOM{Type: NodeTypeElement, Element: "p", WindowCtx: a}
	require.NoError(t, parent.checkCrossOriginMutation(sibling))
	parent.appendChildNode(sibling)
	assert.Len(t, parent.Children, 1)

	// A node from the *same* window context as another window's node is fine.
	bChild := &NodeDOM{Type: NodeTypeElement, Element: "i", WindowCtx: b}
	bParent := &NodeDOM{Type: NodeTypeElement, Element: "div", WindowCtx: b, Children: []*NodeDOM{}}
	require.NoError(t, bParent.checkCrossOriginMutation(bChild))
}

// TestCheckCrossOriginMutationNilCtx keeps the permissive nil contract used by
// Go-level (non-JS) DOM builders.
func TestCheckCrossOriginMutationNilCtx(t *testing.T) {
	parent := &NodeDOM{Type: NodeTypeElement, Element: "div"}
	child := &NodeDOM{Type: NodeTypeElement, Element: "span", WindowCtx: NewWindowContext(&Settings{}, &assets.BuildInfo{}, profiler.CreateProfiler())}
	require.NoError(t, parent.checkCrossOriginMutation(child))
}
