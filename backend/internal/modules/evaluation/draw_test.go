package evaluation

import "testing"

func TestShuffleStringsPermutation(t *testing.T) {
	t.Parallel()
	src := []string{"a", "b", "c", "d", "e"}
	got := append([]string(nil), src...)
	if err := shuffleStrings(got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(src) {
		t.Fatalf("len %d", len(got))
	}
	seen := map[string]int{}
	for _, id := range got {
		seen[id]++
	}
	for _, id := range src {
		if seen[id] != 1 {
			t.Fatalf("id %s count %d", id, seen[id])
		}
	}
}

func TestMergeDrawOrder(t *testing.T) {
	t.Parallel()
	current := []string{"a", "b", "c", "d"}
	got, err := mergeDrawOrder(current, []string{"c", "a"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"c", "a", "b", "d"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if _, err := mergeDrawOrder(current, []string{"a", "x"}); err == nil {
		t.Fatal("unknown id")
	}
	if _, err := mergeDrawOrder(current, []string{"a", "a"}); err == nil {
		t.Fatal("duplicate")
	}
}
