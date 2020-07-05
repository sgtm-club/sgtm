package sgtm

import (
	"fmt"
	"net/http"
)

func httpAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	fmt.Fprintf(w, "yo%s !\n", code)
}
