package challenges

import "testing"

func TestCopySlug(t *testing.T) {
	t.Parallel()
	if got := copySlug("Летняя школа", 1); got != "letnyaya-shkola-copy" {
		t.Fatalf("first copy: %q", got)
	}
	if got := copySlug("Летняя школа", 2); got != "letnyaya-shkola-copy-2" {
		t.Fatalf("second copy: %q", got)
	}
	if got := copySlug("   ", 1); got != "challenge-copy" {
		t.Fatalf("empty title: %q", got)
	}
}
