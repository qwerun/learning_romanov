package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

func checkTable(tables []Table, table string) string {
	for _, v := range tables {
		if table == v.Name {
			return v.Name
		}
	}
	return ""
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	tables, err := GetDbInfo(db)

	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		pathParts := strings.Split(r.URL.Path, "/")

		if strings.HasPrefix(r.URL.Path, "/") && len(pathParts) == 2 && pathParts[1] != "" {
			resCh := checkTable(tables, pathParts[1])
			if resCh == "" {
				err := fmt.Errorf("Table '%v' not found!", pathParts[1])
				_ = writeErrJson(w, err, http.StatusInternalServerError)
				return
			}
			query := r.URL.Query()
			limit, err := strconv.Atoi(query.Get("limit"))
			if err != nil {
				limit = 5
			}
			offset, err := strconv.Atoi(query.Get("offset"))
			if err != nil {
				offset = 0
			}

			req, err := db.Query(fmt.Sprintf("select * from %s limit %v offset %v;", pathParts[1], limit, offset))
			if err != nil {
				_ = writeErrJson(w, err, http.StatusInternalServerError)
				return
			}
			defer req.Close()

			//columns, err := req.Columns()
			//if err != nil {
			//	_ = writeErrJson(w, err, http.StatusInternalServerError)
			//	return
			//}

			fmt.Println(req)
			return
		}
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
	}), nil
}
