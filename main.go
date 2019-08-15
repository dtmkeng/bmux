package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dtmkeng/bmux/bmux"
)

type student struct {
	Name string
	Age  int
}

func main() {
	r := bmux.NewRouter()
	fmt.Println("Server start at 8000...")
	// a := mux.NewRouter()
	log.Fatal(http.ListenAndServe(":8000", r))

	// r := bmux
}

// func home(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "Hello World")
// }
