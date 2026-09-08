// Package component - Virtual DOM reconciler (diff algorithm).
// This runs on the WASM (browser) side to compute minimal DOM patches.
package component

// PatchType describes the kind of DOM operation to perform.
type PatchType int

const (
	PatchCreate  PatchType = iota // Create a new DOM node
	PatchRemove                   // Remove a DOM node
	PatchReplace                  // Replace old node with new node
	PatchUpdate                   // Update attributes/props in place
	PatchText                     // Update text content
	PatchReorder                  // Reorder children (key-based)
)

// Patch represents a single DOM operation produced by the reconciler.
type Patch struct {
	Type    PatchType
	OldNode *Node
	NewNode *Node
	Index   int    // Child index where the patch applies
	Key     string // Key for keyed diffing
}

// Reconcile computes the minimal set of patches to transform oldTree into newTree.
// It returns a flat list of Patch operations.
func Reconcile(old, new *Node) []Patch {
	var patches []Patch
	diff(&patches, old, new, 0)
	return patches
}

func diff(patches *[]Patch, old, new *Node, index int) {
	// Case 1: no old node → create new
	if old == nil {
		*patches = append(*patches, Patch{Type: PatchCreate, NewNode: new, Index: index})
		return
	}

	// Case 2: no new node → remove old
	if new == nil {
		*patches = append(*patches, Patch{Type: PatchRemove, OldNode: old, Index: index})
		return
	}

	// Case 3: both are text nodes
	if old.Type == NodeTypeText && new.Type == NodeTypeText {
		if old.Text != new.Text {
			*patches = append(*patches, Patch{Type: PatchText, OldNode: old, NewNode: new, Index: index})
		}
		return
	}

	// Case 4: different types or different tags → replace
	if old.Type != new.Type || old.Tag != new.Tag {
		*patches = append(*patches, Patch{Type: PatchReplace, OldNode: old, NewNode: new, Index: index})
		return
	}

	// Case 5: same element type → diff props and recurse into children
	if !propsEqual(old.Props, new.Props) {
		*patches = append(*patches, Patch{Type: PatchUpdate, OldNode: old, NewNode: new, Index: index})
	}

	diffChildren(patches, old.Children, new.Children)
}

func diffChildren(patches *[]Patch, oldCh, newCh []*Node) {
	// Key-based diffing for stable lists
	oldKeyed := keyedMap(oldCh)
	newKeyed := keyedMap(newCh)

	maxLen := len(oldCh)
	if len(newCh) > maxLen {
		maxLen = len(newCh)
	}

	for i := 0; i < maxLen; i++ {
		var o, n *Node
		if i < len(oldCh) {
			o = oldCh[i]
		}
		if i < len(newCh) {
			n = newCh[i]
		}

		// Key-based matching
		if n != nil && n.Key != "" {
			if matched, ok := oldKeyed[n.Key]; ok {
				diff(patches, matched, n, i)
				continue
			}
		}
		if o != nil && o.Key != "" {
			if _, stillExists := newKeyed[o.Key]; !stillExists {
				diff(patches, o, nil, i)
				continue
			}
		}

		diff(patches, o, n, i)
	}
}

func keyedMap(nodes []*Node) map[string]*Node {
	m := make(map[string]*Node)
	for _, n := range nodes {
		if n != nil && n.Key != "" {
			m[n.Key] = n
		}
	}
	return m
}

func propsEqual(a, b Props) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		// Simple comparison; func values (event handlers) are never equal
		// so we treat them as always changed — acceptable for now.
		if av != bv {
			return false
		}
	}
	return true
}
