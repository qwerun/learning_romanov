package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomeHandler)
	mux.HandleFunc("/foo", FooHandler)

	middlewares := []func(header http.Handler) http.Handler{
		MyMiddlware,
		SecondMiddlware,
	}
	handler := http.Handler(mux)

	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	err := http.ListenAndServe(":3000", handler)
	if err != nil {
		panic(err)
	}
}

func MyMiddlware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("1 Before")
		handler.ServeHTTP(w, r)
		fmt.Println("1 After")
	})
}

func SecondMiddlware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("2 Before")
		handler.ServeHTTP(w, r)
		fmt.Println("2 After")
	})
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("-- Home")
	w.Write([]byte("This is home page"))
}

func FooHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("-- Foo")
	w.Write([]byte("Foo handler"))
}
