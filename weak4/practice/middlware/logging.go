package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomeHandler)
	mux.HandleFunc("/foo", FooHandler)

	middlewares := []func(header http.Handler) http.Handler{
		LoggingMiddlware,
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

type MyResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (w *MyResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.StatusCode = statusCode
}

func LoggingMiddlware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w2 := &MyResponseWriter{ResponseWriter: w, StatusCode: http.StatusOK}
		handler.ServeHTTP(w2, r)
		log.Printf("%s: [status:%d]\n", r.RequestURI, w2.StatusCode)
	})
}

func SecondMiddlware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		handler.ServeHTTP(w, r)

	})
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("This is home page"))
}

func FooHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Foo handler"))
}
