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

type CR map[string]interface{}

type rowScanner interface {
	Scan(dest ...any) error
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	explorer := &dbExplorer{
		db: db,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(r.URL.Path, "/")
		explorer.pathParts = pathParts
		//if r.Method == http.MethodPost && len(pathParts) == 3 && pathParts[1] != "" && pathParts[2] != "" { //POST /$table/$id
		//	explorer.handlePostById(w, r)
		//}

		if r.URL.Path == "/" {
			explorer.handleGetAllTables(w, r) // GET /
			return
		}

		if len(pathParts) == 2 && pathParts[1] != "" { //GET /$table?limit=5&offset=7
			explorer.handleGetTableData(w, r)
			return
		}

		if len(pathParts) == 3 && pathParts[1] != "" && pathParts[2] != "" { //GET /$table/$id
			explorer.handleGetById(w, r)
			return
		}

		http.NotFound(w, r)
	}), nil
}

func writeJSON(w http.ResponseWriter, v any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func writeErrJSON(w http.ResponseWriter, err error, status int) error {
	result := map[string]any{
		"ok":    false,
		"error": err.Error(),
	}
	return writeJSON(w, result, status)
}

func dbRowToMap(scanner rowScanner, columns []Column) (CR, error) {
	row := CR{}
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))

	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := scanner.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	for i, col := range columns {
		val := values[i]
		switch v := val.(type) {
		case nil:
			row[col.Name] = nil
		case []byte:
			s := string(v)
			if iVal, err := strconv.Atoi(s); err == nil {
				row[col.Name] = iVal
			} else {
				row[col.Name] = s
			}
		default:
			row[col.Name] = v
		}
	}
	return row, nil
}

func (explorer *dbExplorer) getDbInfo() error {
	if explorer.cacheLoaded {
		return explorer.cacheErr
	}

	tableRows, err := explorer.db.Query("SHOW TABLES;")
	if err != nil {
		explorer.cacheErr = err
		return err
	}
	defer tableRows.Close()

	var tables []Table
	for tableRows.Next() {
		var table Table
		if err := tableRows.Scan(&table.Name); err != nil {
			explorer.cacheErr = err
			return err
		}

		columnRows, err := explorer.db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`;", table.Name))
		if err != nil {
			explorer.cacheErr = err
			return err
		}
		defer columnRows.Close()

		for columnRows.Next() {
			var col Column
			var empty sql.RawBytes
			if err := columnRows.Scan(&col.Name, &empty, &empty, &empty, &empty, &empty, &empty, &empty, &empty); err != nil {
				explorer.cacheErr = err
				return err
			}
			table.Columns = append(table.Columns, col)
		}
		tables = append(tables, table)
	}

	explorer.cache = tables
	explorer.cacheLoaded = true
	return nil
}

func (explorer *dbExplorer) checkTable(table string) string {
	for _, t := range explorer.cache {
		if table == t.Name {
			return t.Name
		}
	}
	return ""
}

func (explorer *dbExplorer) parseId() (int, error) {
	idStr := explorer.pathParts[2]
	id, err := strconv.Atoi(idStr)
	return id, err
}

func (explorer *dbExplorer) getColumns(tableName string) []Column {
	var columns []Column
	for _, t := range explorer.cache {
		if t.Name == tableName {
			columns = t.Columns
			break
		}
	}
	return columns
}

func (explorer *dbExplorer) handleGetAllTables(w http.ResponseWriter, r *http.Request) {
	err := explorer.getDbInfo()
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}
	var res []string
	for _, t := range explorer.cache {
		res = append(res, t.Name)
	}
	response := CR{
		"response": CR{
			"tables": res,
		},
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
	}
}

func (explorer *dbExplorer) handleGetTableData(w http.ResponseWriter, r *http.Request) {
	err := explorer.getDbInfo()
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}
	tableName := explorer.pathParts[1]
	if explorer.checkTable(tableName) == "" {
		_ = writeJSON(w, CR{"error": "unknown table"}, http.StatusNotFound)
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

	req, err := explorer.db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?;", tableName), limit, offset)
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}
	defer req.Close()
	columns := explorer.getColumns(tableName)

	var result []CR
	for req.Next() {
		row, err := dbRowToMap(req, columns)
		if err != nil {
			_ = writeErrJSON(w, err, http.StatusInternalServerError)
			return
		}
		result = append(result, row)
	}

	response := CR{
		"response": CR{
			"records": result,
		},
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
	}
}

func (explorer *dbExplorer) handleGetById(w http.ResponseWriter, r *http.Request) {
	err := explorer.getDbInfo()
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}

	tableName := explorer.pathParts[1]

	if explorer.checkTable(tableName) == "" {
		_ = writeJSON(w, CR{"error": "unknown table"}, http.StatusNotFound)
		return
	}
	id, err := explorer.parseId()
	if err != nil {
		_ = writeErrJSON(w, fmt.Errorf("Invalid ID format: %v", err), http.StatusBadRequest)
		return
	}

	req := explorer.db.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE id = ?;", tableName), id)

	columns := explorer.getColumns(tableName)

	var result []CR

	row, err := dbRowToMap(req, columns)
	if err != nil {
		_ = writeJSON(w, CR{"error": "record not found"}, http.StatusNotFound)
		return
	}
	result = append(result, row)

	response := CR{
		"response": CR{
			"records": result,
		},
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
	}
}

//
//func (explorer *dbExplorer) handlePostById(w http.ResponseWriter, r *http.Request) {
//	err := explorer.getDbInfo()
//	if err != nil {
//		_ = explorer.writeErrJSON(w, err, http.StatusInternalServerError)
//		return
//	}
//
//	tableName := explorer.pathParts[1]
//
//	if explorer.checkTable(tableName) == "" {
//		_ = explorer.writeErrJSON(w, fmt.Errorf("Table '%s' not found", tableName), http.StatusNotFound)
//		return
//	}
//	id, err := explorer.parseId()
//	if err != nil {
//		_ = explorer.writeErrJSON(w, fmt.Errorf("Invalid ID format: %v", err), http.StatusBadRequest)
//		return
//	}
//
//	req := explorer.db.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE id = ?;", tableName), id)
//
//	columns := explorer.getColumns(tableName)
//
//	result, err := explorer.dbRowToMap(req, columns)
//	if err != nil {
//		_ = explorer.writeErrJSON(w, err, http.StatusInternalServerError)
//		return
//	}
//
//	if err := explorer.writeJSON(w, result, http.StatusOK); err != nil {
//		_ = explorer.writeErrJSON(w, err, http.StatusInternalServerError)
//	}
//}
