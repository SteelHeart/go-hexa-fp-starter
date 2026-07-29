package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestMapAndFilterPreserveOrder: item order is preserved.
//
// Cursor pagination rests entirely on a stable order: a function silently
// reordering would make rows skipped or repeated between two pages, and the
// symptom would only show past the first page.
func TestMapAndFilterPreserveOrder(t *testing.T) {
	t.Parallel()

	source := []int{3, 1, 4, 1, 5}

	transformed := fp.Map(source, double)
	want := []int{6, 2, 8, 2, 10}
	for i, w := range want {
		if transformed[i] != w {
			t.Fatalf("Map = %v, want %v", transformed, want)
		}
	}

	kept := fp.Filter([]int{1, 2, 3, 4, 5, 6}, even)
	wantFiltered := []int{2, 4, 6}
	if len(kept) != len(wantFiltered) {
		t.Fatalf("Filter = %v, want %v", kept, wantFiltered)
	}
	for i, w := range wantFiltered {
		if kept[i] != w {
			t.Fatalf("Filter = %v, want %v", kept, wantFiltered)
		}
	}
}
