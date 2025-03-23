package main //Незакончен Незакончен Незакончен Незакончен Незакончен 7.4

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type MResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

type User struct {
	Id   int
	Name string
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomesHandler)
	mux.HandleFunc("/getme", GetMeHandler)

	middlewares := []func(header http.Handler) http.Handler{
		LogginMiddlware,
		AuthMiddlware,
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

func (w *MResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.StatusCode = statusCode
}

func LogginMiddlware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w2 := &MResponseWriter{ResponseWriter: w, StatusCode: http.StatusOK}
		handler.ServeHTTP(w2, r)
		log.Printf("%s: [status:%d]\n", r.RequestURI, w2.StatusCode)
	})
}

func AuthMiddlware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})
}

func WriteJSON(w io.Writer, v any) {
	bytes, _ := json.Marshal(v)
	w.Write(bytes)
}

func HomesHandler(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, map[string]any{
		"ok": true,
	})
}

func GetMeHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	WriteJSON(w, map[string]any{
		"ok":   true,
		"user": user,
	})
}
