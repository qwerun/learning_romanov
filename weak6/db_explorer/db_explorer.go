package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

func GetDbInfo(db *sql.DB) ([]Table, error) {
	var tables []Table
	tableRows, err := db.Query("SHOW TABLES;")
	if err != nil {
		return nil, err
	}
	defer tableRows.Close()

	for tableRows.Next() {
		var table Table
		err := tableRows.Scan(&table.Name)
		if err != nil {
			return nil, err
		}
		columnRows, err := db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`;", table.Name))
		if err != nil {
			return nil, err
		}
		defer columnRows.Close()
		for columnRows.Next() {
			var columns Column
			var empty sql.RawBytes

			err := columnRows.Scan(&columns.Name, &empty, &empty, &empty, &empty, &empty, &empty, &empty, &empty)

			if err != nil {
				return nil, err
			}

			table.Columns = append(table.Columns, columns)
		}

		tables = append(tables, table)
	}

	return tables, nil
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	tables, err := GetDbInfo(db)

	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			var res []string
			for _, v := range tables {
				res = append(res, v.Name)
			}
			err = writeJson(w, res, http.StatusOK)
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
