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
	Name  string `json:"name"`
	Types string `json:"types"`
	Null  string `json:"null"`
}

type dbExplorer struct {
	db          *sql.DB
	tableInfo   []Table
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
		if r.Method == http.MethodPut && len(pathParts) == 3 && pathParts[1] != "" && pathParts[2] == "" {
			explorer.handlePut(w, r) // PUT /$table/
			return
		}

		if r.Method == http.MethodPost && len(pathParts) == 3 && pathParts[1] != "" && pathParts[2] != "" {
			explorer.handlePostById(w, r) //POST /$table/$id
			return
		}
		if r.URL.Path == "/" {
			explorer.handleGetAllTables(w, r) // GET /
			return
		}
		if len(pathParts) == 2 && pathParts[1] != "" {
			explorer.handleGetTableData(w, r) //GET /$table?limit=5&offset=7
			return
		}
		if len(pathParts) == 3 && pathParts[1] != "" && pathParts[2] != "" {
			explorer.handleGetById(w, r) //GET /$table/$id
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
			} else if fVal, err := strconv.ParseFloat(s, 64); err == nil {
				row[col.Name] = fVal
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
			if err := columnRows.Scan(&col.Name, &col.Types, &empty, &col.Null, &empty, &empty, &empty, &empty, &empty); err != nil {
				explorer.cacheErr = err
				return err
			}
			table.Columns = append(table.Columns, col)

		}
		tables = append(tables, table)
	}

	explorer.tableInfo = tables
	explorer.cacheLoaded = true
	return nil
}

func (explorer *dbExplorer) checkTable(table string) string {
	for _, t := range explorer.tableInfo {
		if table == t.Name {
			return t.Name
		}
	}
	return ""
}

func (explorer *dbExplorer) checkBody(tableName string, updates CR) CR {
	var tableInfo Table
	for _, v := range explorer.tableInfo {
		if v.Name == tableName {
			tableInfo = v
		}
	}

	for columnName, value := range updates {
		var colDef *Column
		for _, col := range tableInfo.Columns {
			if col.Name == columnName {
				colDef = &col
				break
			}
		}
		if colDef == nil {
			return CR{"error": fmt.Sprintf("unknown column %s", columnName)}
		}

		if colDef.Null == "NO" && value == nil {
			return CR{"error": fmt.Sprintf("field %s have invalid type", columnName)}
		}

		if strings.HasPrefix(colDef.Types, "varchar") || colDef.Types == "text" {
			if _, ok := value.(string); !ok {
				return CR{"error": fmt.Sprintf("field %s have invalid type", columnName)}
			}
		}

		if strings.HasPrefix(colDef.Types, "int") {
			if _, ok := value.(float64); !ok {
				return CR{"error": fmt.Sprintf("field %s have invalid type", columnName)}
			}
		}
	}
	return nil
}

func (explorer *dbExplorer) parseId() (int, error) {
	idStr := explorer.pathParts[2]
	id, err := strconv.Atoi(idStr)
	return id, err
}

func (explorer *dbExplorer) getColumns(tableName string) []Column {
	var columns []Column
	for _, t := range explorer.tableInfo {
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
	for _, t := range explorer.tableInfo {
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

func (explorer *dbExplorer) handlePostById(w http.ResponseWriter, r *http.Request) {
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

	var updates CR
	err = json.NewDecoder(r.Body).Decode(&updates)
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	if updates["id"] != nil {
		_ = writeJSON(w, CR{"error": "field id have invalid type"}, http.StatusBadRequest)
		return
	}
	errMessege := explorer.checkBody(tableName, updates)
	if err != nil {
		_ = writeJSON(w, errMessege, http.StatusBadRequest)
	}

	setParts := make([]string, 0, len(updates))
	params := make([]interface{}, 0, len(updates)+1)

	for columnName, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = ?", columnName))
		params = append(params, value)
	}

	params = append(params, id)

	sqlQuery := fmt.Sprintf(`
        UPDATE %s 
        SET %s 
        WHERE id = ?;
    `, tableName, strings.Join(setParts, ", "))

	result, err := explorer.db.Exec(sqlQuery, params...)
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}

	response := CR{
		"updated": rowsAffected,
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
	}
	return
}

func (explorer *dbExplorer) handlePut(w http.ResponseWriter, r *http.Request) {
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

	var updates CR
	err = json.NewDecoder(r.Body).Decode(&updates)
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	errMessege := explorer.checkBody(tableName, updates)
	if err != nil {
		_ = writeJSON(w, errMessege, http.StatusBadRequest)
	}

	columns := make([]string, 0, len(updates))
	placeholders := make([]string, 0, len(updates))
	params := make([]interface{}, 0, len(updates))

	for col, val := range updates {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		params = append(params, val)
	}

	sqlQuery := fmt.Sprintf(`
        INSERT INTO %s (%s)
        VALUES (%s);
    `, tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	result, err := explorer.db.Exec(sqlQuery, params...)
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
		return
	}

	response := CR{
		"id": lastId,
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeErrJSON(w, err, http.StatusInternalServerError)
	}
	return
}
