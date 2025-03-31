package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type Table struct {
	Name    string `json:"name"`
	Columns []Column
}

type Column struct {
	Name string `json:"name"`
}

func writeJson(w http.ResponseWriter, v any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func writeErrJson(w http.ResponseWriter, err error, status int) error {
	result := map[string]any{
		"ok":    false,
		"error": err.Error(),
	}
	return writeJson(w, result, status)
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			var tables []Table
			rows, err := db.Query("SHOW TABLES;")
			if err != nil {
				_ = writeErrJson(w, err, http.StatusInternalServerError)
				return
			}

			for rows.Next() {
				var table Table
				err := rows.Scan(&table.Name)
				if err != nil {
					_ = writeErrJson(w, err, http.StatusInternalServerError)
					return
				}
				tables = append(tables, table)
			}

			err = writeJson(w, tables, http.StatusOK)
			if err != nil {
				_ = writeErrJson(w, err, http.StatusInternalServerError)
				return
			}
			return
		}

		if r.URL.Path == "/roma" {
			w.Write([]byte("roma!"))
			return
		}

	}), nil
}
