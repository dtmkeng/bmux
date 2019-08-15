package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/dtmkeng/bmux/bmux"
)

type student struct {
	Name string
	Age  int
}

var port = 8080

func main() {
	r := bmux.NewRouter()
	// fmt.Println("Server start at 8000...")
	// a := mux.NewRouter()
	r.Get("/", func(ctx bmux.Context) error {
		return ctx.String("hello")
	})
	// r.Run() http serer
	fmt.Println("Server start at ... ", port)
	// color.GreedString()
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), r)) // defaut Listener and serve

	// r := bmux
}

// func home(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "Hello World")
// }
