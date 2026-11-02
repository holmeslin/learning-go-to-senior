package signup_test

import (
	"errors"
	"slices"
	"testing"

	signup "github.com/holmeslin/learning-go-to-senior/examples/chapter13/01_design_before_details/before"
)

func TestStoreCreateAndGet(t *testing.T) {
	store := signup.NewStore(&fakeNotifier{})
	created, err := store.Create("go-meetup", 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := created.Register("u1"); err != nil {
		t.Fatal(err)
	}

	got, ok := store.Get("go-meetup")
	if !ok {
		t.Fatal("Get(go-meetup) not found")
	}
	if want := []string{"u1"}; !slices.Equal(got.Confirmed(), want) {
		t.Errorf("Confirmed() = %v, want %v", got.Confirmed(), want)
	}
	if _, ok := store.Get("missing"); ok {
		t.Error("Get(missing) found an event")
	}
	if _, err := store.Create("go-meetup", 5); !errors.Is(err, signup.ErrEventExists) {
		t.Errorf("duplicate Create = %v, want ErrEventExists", err)
	}
}
