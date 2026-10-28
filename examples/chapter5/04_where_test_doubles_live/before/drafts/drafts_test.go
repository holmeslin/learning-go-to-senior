package drafts

import "testing"

func TestRestoreReturnsAutosavedText(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	if err := Autosave(store, "plan", "outline"); err != nil {
		t.Fatalf("Autosave() error = %v", err)
	}

	got, err := Restore(store, "plan")
	if err != nil || got != "outline" {
		t.Errorf("Restore() = %q, %v; want %q, nil", got, err, "outline")
	}
}

func TestRestoreWithoutDraftIsEmpty(t *testing.T) {
	t.Parallel()
	got, err := Restore(newFakeStore(), "plan")
	if err != nil || got != "" {
		t.Errorf("Restore() = %q, %v; want empty string, nil", got, err)
	}
}
