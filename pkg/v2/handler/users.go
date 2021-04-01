package handler

import (
	"fmt"

	userstore "moul.io/sgtm/pkg/v2/store"
)

// fixme: this will be Func(w,r) and will handle the res on some /route/
func GetUser() {
	// validate req params
	// create new instance of the Store(storage interface)

	store := userstore.NewStore(nil)

	user, err := store.GetUser(0)
	if err != nil {
		panic(err)
	}

	// Write user to json response
	fmt.Print(user)

}
