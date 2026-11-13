package im

import "testing"

// TestMapHashCollisions forces full-hash collisions without depending on random hash seeds.
func TestMapHashCollisions(t *testing.T) {
	original := mapLeaf[int, string](7, 1, "one")
	second, added := setMapNode(original, 0, 7, 2, "two")
	if !added || len(original.entries) != 1 || len(second.entries) != 2 {
		t.Fatal("collision insertion changed the original bucket")
	}
	replaced, added := setMapNode(second, 0, 7, 1, "changed")
	if added {
		t.Fatal("replacement added a key")
	}
	if got, ok := getMapNode(replaced, 7, 1); !ok || got != "changed" {
		t.Fatal("collision replacement failed")
	}
	if got, ok := getMapNode(second, 7, 1); !ok || got != "one" {
		t.Fatal("collision replacement changed the previous bucket")
	}
	if _, ok := getMapNode(second, 7, 3); ok {
		t.Fatal("missing collision key was found")
	}
	deleted, removed := deleteMapNode(replaced, 0, 7, 2)
	if !removed || len(deleted.entries) != 1 || len(replaced.entries) != 2 {
		t.Fatal("collision deletion changed the previous bucket")
	}
	if unchanged, removed := deleteMapNode(deleted, 0, 7, 99); removed || unchanged != deleted {
		t.Fatal("missing collision deletion changed storage")
	}
	if unchanged, removed := deleteMapNode(deleted, 0, 8, 1); removed || unchanged != deleted {
		t.Fatal("different hash deletion changed storage")
	}
	if empty, removed := deleteMapNode(deleted, 0, 7, 1); !removed || empty != nil {
		t.Fatal("last collision deletion did not remove the leaf")
	}
}

// TestMapDeepHashPaths checks differing high bits, singleton branch collapse, and reinsertion.
func TestMapDeepHashPaths(t *testing.T) {
	root := mapLeaf[int, int](0, 1, 10)
	root, _ = setMapNode(root, 0, 1<<63, 2, 20)
	root, _ = setMapNode(root, 0, 1<<62, 3, 30)
	for _, entry := range []struct {
		hash uint64
		key  int
	}{{0, 1}, {1 << 63, 2}, {1 << 62, 3}} {
		if value, ok := getMapNode(root, entry.hash, entry.key); !ok || value != entry.key*10 {
			t.Fatalf("deep key %d is missing", entry.key)
		}
	}
	first, removed := deleteMapNode(root, 0, 1<<63, 2)
	if !removed {
		t.Fatal("deep deletion failed")
	}
	second, removed := deleteMapNode(first, 0, 1<<62, 3)
	if !removed || second.entries == nil || len(second.entries) != 1 {
		t.Fatal("singleton leaf did not collapse")
	}
	third, added := setMapNode(second, 0, 1<<63, 4, 40)
	if !added {
		t.Fatal("insertion after collapse failed")
	}
	if value, ok := getMapNode(third, 1<<63, 4); !ok || value != 40 {
		t.Fatal("inserted key after collapse is missing")
	}
	if value, ok := getMapNode(root, 1<<63, 2); !ok || value != 20 {
		t.Fatal("collapsed path changed the original")
	}
}

// TestMapSharesUnchangedBranches checks that a point update retains unrelated child pointers.
func TestMapSharesUnchangedBranches(t *testing.T) {
	root := mapLeaf[int, int](0, 1, 10)
	root, _ = setMapNode(root, 0, 1, 2, 20)
	updated, _ := setMapNode(root, 0, 1, 2, 99)
	if root == updated || root.children[0] != updated.children[0] || root.children[1] == updated.children[1] {
		t.Fatal("update did not copy only the changed branch")
	}
}
