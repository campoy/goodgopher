package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe("0.0.0.0:4321", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello")
}
