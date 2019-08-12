package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/dtmkeng/project/bmux/bmux"
)

type student struct {
	Name string
	Age  int
}

var port = 8080

func main() {
	r := bmux.NewRouter()
	r.Get("/hello", home)
	fmt.Println("Server start at 8000...")
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), r))
}

func home(ctx bmux.Context) error {
	return nil
}
