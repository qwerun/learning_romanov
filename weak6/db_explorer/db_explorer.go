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
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

type Column struct {
	Name string `json:"name"`
}

type dbExplorer struct {
	db          *sql.DB
	cache       []Table
	pathParts   []string
	cacheErr    error
	cacheLoaded bool
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	explorer := &dbExplorer{
		db: db,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(r.URL.Path, "/")
		explorer.pathParts = pathParts
		if r.URL.Path == "/" {
			explorer.handleGetAllTables(w, r) // GET /
			return
		}

		if len(pathParts) == 2 && pathParts[1] != "" { //GET /$table?limit=5&offset=7
			explorer.handleGetTableData(w, r)
			return
		}

		http.NotFound(w, r)
	}), nil
}

func (explorer *dbExplorer) writeJSON(w http.ResponseWriter, v any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func (explorer *dbExplorer) writeErrJSON(w http.ResponseWriter, err error, status int) error {
	result := map[string]any{
		"ok":    false,
		"error": err.Error(),
	}
	return explorer.writeJSON(w, result, status)
}

func (explorer *dbExplorer) GetDbInfo() ([]Table, error) {
	if explorer.cacheLoaded {
		return explorer.cache, explorer.cacheErr
	}

	tableRows, err := explorer.db.Query("SHOW TABLES;")
	if err != nil {
		explorer.cacheErr = err
		return nil, err
	}
	defer tableRows.Close()

	var tables []Table
	for tableRows.Next() {
		var table Table
		if err := tableRows.Scan(&table.Name); err != nil {
			explorer.cacheErr = err
			return nil, err
		}

		columnRows, err := explorer.db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`;", table.Name))
		if err != nil {
			explorer.cacheErr = err
			return nil, err
		}
		defer columnRows.Close()

		for columnRows.Next() {
			var col Column
			var empty sql.RawBytes
			if err := columnRows.Scan(&col.Name, &empty, &empty, &empty, &empty, &empty, &empty, &empty, &empty); err != nil {
				explorer.cacheErr = err
				return nil, err
			}
			table.Columns = append(table.Columns, col)
		}
		tables = append(tables, table)
	}

	explorer.cache = tables
	explorer.cacheLoaded = true
	return explorer.cache, nil
}

func (explorer *dbExplorer) checkTable(tables []Table, table string) string {
	for _, t := range tables {
		if table == t.Name {
			return t.Name
		}
	}
	return ""
}

func (explorer *dbExplorer) handleGetAllTables(w http.ResponseWriter, r *http.Request) {
	tables, err := explorer.GetDbInfo()
	if err != nil {
		_ = explorer.writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}
	var res []string
	for _, t := range tables {
		res = append(res, t.Name)
	}
	if err := explorer.writeJSON(w, res, http.StatusOK); err != nil {
		_ = explorer.writeErrJSON(w, err, http.StatusInternalServerError)
	}
}

func (explorer *dbExplorer) handleGetTableData(w http.ResponseWriter, r *http.Request) {
	tables, err := explorer.GetDbInfo()
	if err != nil {
		_ = explorer.writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}
	tableName := explorer.pathParts[1]
	if explorer.checkTable(tables, tableName) == "" {
		_ = explorer.writeErrJSON(w, fmt.Errorf("Table '%s' not found", tableName), http.StatusNotFound)
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

	req, err := explorer.db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d;", tableName, limit, offset))
	if err != nil {
		_ = explorer.writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}
	defer req.Close()

	var columns []Column
	for _, t := range tables {
		if t.Name == tableName {
			columns = t.Columns
			break
		}
	}

	var result []map[string]any
	for req.Next() {
		row := make(map[string]any)
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := req.Scan(valuePtrs...); err != nil {
			_ = explorer.writeErrJSON(w, err, http.StatusInternalServerError)
			return
		}

		for i, col := range columns {
			switch v := values[i].(type) {
			case []byte:
				row[col.Name] = string(v)
			default:
				row[col.Name] = v
			}
		}
		result = append(result, row)
	}

	if err := explorer.writeJSON(w, result, http.StatusOK); err != nil {
		_ = explorer.writeErrJSON(w, err, http.StatusInternalServerError)
	}
}
