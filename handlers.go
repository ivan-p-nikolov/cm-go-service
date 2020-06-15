package main

import (
	"net/http"
)

func TestHandler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Hello there from test handler!"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
