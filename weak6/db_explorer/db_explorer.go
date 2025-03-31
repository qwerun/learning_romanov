package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type Table struct {
	Name string `json:"name"`
}

func jsonResponse(r *http.ResponseWriter, text any, st int) {

}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/" {
			var tables []Table
			rows, err := db.Query("SHOW TABLES;")
			if err != nil {
				result := map[string]any{
					"ok":    false,
					"error": err.Error(),
				}
				json.NewEncoder(w).Encode(result)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			for rows.Next() {
				var table Table
				err := rows.Scan(&table.Name)
				if err != nil {
					panic(err)
				}
				tables = append(tables, table)
			}

			jsonData, err := json.Marshal(tables)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error encoding JSON"))
				return
			}

			w.Write(jsonData)
			return
		}

		if r.URL.Path == "/roma" {
			w.Write([]byte("roma!"))
			return
		}

	}), nil
	return nil, nil
}
