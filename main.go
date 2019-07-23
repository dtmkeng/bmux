package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dtmkeng/project/bmux/bmux"
)

type student struct {
	Name string
	Age  int
}

func main() {
	r := bmux.NewRouter()
	r.HandleFunc("/", home)
	fmt.Println("Server start at 8000...")
	log.Fatal(http.ListenAndServe(":8000", r))
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World")
}
