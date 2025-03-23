package main

import (
	"encoding/json"
	"net/http"
)

type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func main() {

	http.HandleFunc("/user", UserHandler)

	http.ListenAndServe(":3000", nil)

}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user1 := User{Id: 1, Name: "Roman"}
	bytes, err := json.Marshal(user1)
	if err != nil {
		result := map[string]any{"ok": false,
			"error": "500 Internal Server Error: " + err.Error()}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(result)
		return
	}
	w.Write(bytes)
}
