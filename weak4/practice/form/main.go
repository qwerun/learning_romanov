package main

import (
	"io"
	"net/http"
)

// curl -X POST -F foo=123 localhost:3000/form
func main() {
	http.HandleFunc("/form", FormHandle)
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		panic(err)
	}
}

func FormHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		io.WriteString(w, "405 Method Not Allowed")
		return
	}

	foo := r.FormValue("foo")
	w.Write([]byte("OK: " + foo))
}
