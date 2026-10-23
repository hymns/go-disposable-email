package disposable_test

import (
	"errors"
	"fmt"

	disposable "github.com/hymns/go-disposable-email"
)

func ExampleCheck() {
	for _, email := range []string{"someone@gmail.com", "someone@mailinator.com"} {
		if err := disposable.Check(email); errors.Is(err, disposable.ErrDisposable) {
			fmt.Println(email, "refused")
		} else {
			fmt.Println(email, "accepted")
		}
	}
	// Output:
	// someone@gmail.com accepted
	// someone@mailinator.com refused
}

func ExampleNew() {
	c := disposable.New(
		disposable.WithBlocked("new-throwaway.example"),
		disposable.WithAllowed("partner.mailinator.com"),
	)
	fmt.Println(c.IsDisposable("a@x.new-throwaway.example"))
	fmt.Println(c.IsDisposable("a@partner.mailinator.com"))
	fmt.Println(c.IsDisposable("a@mailinator.com"))
	// Output:
	// true
	// false
	// true
}
