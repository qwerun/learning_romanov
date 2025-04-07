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
	Name    string `json:"name"`
	Types   string `json:"types"`
	Null    string `json:"null"`
	Default string `json:"default"`
}

type dbExplorer struct {
	db          *sql.DB
	tableInfo   []Table
	pathParts   []string
	cacheErr    error
	cacheLoaded bool
}

type CT map[string]interface{}

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
		if r.Method == http.MethodDelete && len(pathParts) == 3 && pathParts[1] != "" && pathParts[2] != "" {
			explorer.handleDeleteById(w, r) //DELETE /$table/$id
			return
		}
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

func dbRowToMap(scanner rowScanner, columns []Column) (CT, error) {
	row := CT{}
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

	var tableNames []string
	for tableRows.Next() {
		var tableName string
		if err := tableRows.Scan(&tableName); err != nil {
			explorer.cacheErr = err
			return err
		}
		tableNames = append(tableNames, tableName)
	}

	var tables []Table
	for _, tName := range tableNames {
		var table Table
		table.Name = tName

		columnRows, err := explorer.db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`;", tName))
		if err != nil {
			explorer.cacheErr = err
			return err
		}

		for columnRows.Next() {
			var col Column
			var empty sql.RawBytes
			var defaultVal sql.NullString
			if err := columnRows.Scan(&col.Name, &col.Types, &empty, &col.Null, &empty, &defaultVal, &empty, &empty, &empty); err != nil {
				columnRows.Close()
				explorer.cacheErr = err
				return err
			}

			if defaultVal.Valid {
				col.Default = defaultVal.String
			} else {
				col.Default = ""
			}

			table.Columns = append(table.Columns, col)
		}

		if err = columnRows.Close(); err != nil {
			explorer.cacheErr = err
			return err
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

func (explorer *dbExplorer) checkBody(tableName string, updates CT) (CT, CT) {

	var tableInfo Table
	for _, v := range explorer.tableInfo {
		if v.Name == tableName {
			tableInfo = v
		}
	}
	res := make(CT)
	for columnName, value := range updates {
		var colDef *Column
		for _, col := range tableInfo.Columns {
			if col.Name == columnName {
				colDef = &col
				break
			}
		}
		if colDef == nil {
			continue
		}

		if colDef.Null == "NO" && value == nil {
			return nil, CT{"error": fmt.Sprintf("field %s have invalid type", columnName)}
		} else if value == nil {
			continue
		}

		if strings.HasPrefix(colDef.Types, "varchar") || colDef.Types == "text" {
			if _, ok := value.(string); !ok {
				return nil, CT{"error": fmt.Sprintf("field %s have invalid type", columnName)}
			}
		}

		if strings.HasPrefix(colDef.Types, "int") {
			if _, ok := value.(float64); !ok {
				return nil, CT{"error": fmt.Sprintf("field %s have invalid type", columnName)}
			}
		}

		res[columnName] = value
	}

	for _, col := range tableInfo.Columns {
		if _, exists := res[col.Name]; !exists {
			if col.Null == "NO" && col.Default == "" {
				if strings.HasPrefix(col.Types, "varchar") || col.Types == "text" {
					res[col.Name] = ""
				} else if strings.HasPrefix(col.Types, "int") {
					res[col.Name] = 0
				} else {
					res[col.Name] = nil
				}
			}
		}
	}
	return res, nil
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

func (explorer *dbExplorer) getPKColumn(tableName string) (string, error) {
	query := `
		SELECT COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_NAME = ? AND CONSTRAINT_NAME = 'PRIMARY'
		LIMIT 1;
	`
	row := explorer.db.QueryRow(query, tableName)
	var pkColumn string
	if err := row.Scan(&pkColumn); err != nil {
		return "", err
	}
	return pkColumn, nil
}

func (explorer *dbExplorer) handleGetAllTables(w http.ResponseWriter, r *http.Request) {

	err := explorer.getDbInfo()
	if err != nil {
		_ = writeJSON(w, CT{"error": "getDbInfo error"}, http.StatusInternalServerError)
		return
	}
	var res []string
	for _, t := range explorer.tableInfo {
		res = append(res, t.Name)
	}
	response := CT{
		"response": CT{
			"tables": res,
		},
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeJSON(w, CT{"error": "GetAllTables error"}, http.StatusInternalServerError)
	}
}

func (explorer *dbExplorer) handleGetTableData(w http.ResponseWriter, r *http.Request) {
	err := explorer.getDbInfo()
	if err != nil {
		_ = writeJSON(w, CT{"error": "getDbInfo error"}, http.StatusInternalServerError)
		return
	}
	tableName := explorer.pathParts[1]
	if explorer.checkTable(tableName) == "" {
		_ = writeJSON(w, CT{"error": "unknown table"}, http.StatusNotFound)
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
		_ = writeJSON(w, CT{"error": "Query table data error"}, http.StatusInternalServerError)
		return
	}
	defer req.Close()
	columns := explorer.getColumns(tableName)

	var result []CT
	for req.Next() {
		row, err := dbRowToMap(req, columns)
		if err != nil {
			_ = writeJSON(w, CT{"error": "rowToMap error"}, http.StatusInternalServerError)
			return
		}
		result = append(result, row)
	}

	response := CT{
		"response": CT{
			"records": result,
		},
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeJSON(w, CT{"error": "getTableData error"}, http.StatusInternalServerError)
	}
}

func (explorer *dbExplorer) handleGetById(w http.ResponseWriter, r *http.Request) {
	err := explorer.getDbInfo()
	if err != nil {
		_ = writeJSON(w, CT{"error": "getDbInfo error"}, http.StatusInternalServerError)
		return
	}

	tableName := explorer.pathParts[1]

	if explorer.checkTable(tableName) == "" {
		_ = writeJSON(w, CT{"error": "unknown table"}, http.StatusNotFound)
		return
	}
	id, err := explorer.parseId()
	if err != nil {
		_ = writeJSON(w, CT{"error": "parseId error"}, http.StatusBadRequest)
		return
	}
	pkColumn, err := explorer.getPKColumn(tableName)
	if err != nil {
		_ = writeJSON(w, CT{"error": "PKColumn error"}, http.StatusInternalServerError)
		return
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?;", tableName, pkColumn)

	req := explorer.db.QueryRow(query, id)

	columns := explorer.getColumns(tableName)

	var result CT

	row, err := dbRowToMap(req, columns)
	if err != nil {
		_ = writeJSON(w, CT{"error": "record not found"}, http.StatusNotFound)
		return
	}
	result = row

	response := CT{
		"response": CT{
			"record": result,
		},
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeJSON(w, CT{"error": "GetById error"}, http.StatusInternalServerError)
	}
}

func (explorer *dbExplorer) handlePostById(w http.ResponseWriter, r *http.Request) {
	err := explorer.getDbInfo()
	if err != nil {
		_ = writeJSON(w, CT{"error": "getDbInfo error"}, http.StatusInternalServerError)
		return
	}

	tableName := explorer.pathParts[1]

	if explorer.checkTable(tableName) == "" {
		_ = writeJSON(w, CT{"error": "unknown table"}, http.StatusNotFound)
		return
	}
	id, err := explorer.parseId()
	if err != nil {
		_ = writeJSON(w, CT{"error": "parseId table"}, http.StatusBadRequest)
		return
	}

	var updates CT
	err = json.NewDecoder(r.Body).Decode(&updates)
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}
	r.Body.Close()
	pkColumn, err := explorer.getPKColumn(tableName)
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}

	if updates[pkColumn] != nil {
		_ = writeJSON(w, CT{"error": fmt.Sprintf("field %s have invalid type", pkColumn)}, http.StatusBadRequest)
		return
	}
	_, errMessege := explorer.checkBody(tableName, updates)
	if errMessege != nil {
		_ = writeJSON(w, errMessege, http.StatusBadRequest)
		return
	}

	setParts := make([]string, 0, len(updates))
	params := make([]interface{}, 0, len(updates)+1)

	for columnName, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = ?", columnName))
		params = append(params, value)
	}

	params = append(params, id)

	pkColumn, err = explorer.getPKColumn(tableName)
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}

	sqlQuery := fmt.Sprintf(`
        UPDATE %s 
        SET %s 
        WHERE %s = ?;
    `, tableName, strings.Join(setParts, ", "), pkColumn)

	result, err := explorer.db.Exec(sqlQuery, params...)
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}

	response := CT{
		"response": CT{
			"updated": rowsAffected,
		},
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
	}
	return
}

func (explorer *dbExplorer) handlePut(w http.ResponseWriter, r *http.Request) {
	err := explorer.getDbInfo()
	if err != nil {
		_ = writeJSON(w, CT{"error": "getDbInfo error"}, http.StatusInternalServerError)
		return
	}

	tableName := explorer.pathParts[1]

	if explorer.checkTable(tableName) == "" {
		_ = writeJSON(w, CT{"error": "unknown table"}, http.StatusNotFound)
		return
	}

	var updates CT
	err = json.NewDecoder(r.Body).Decode(&updates)
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}
	r.Body.Close()
	updates, errMessege := explorer.checkBody(tableName, updates)
	if errMessege != nil {
		_ = writeJSON(w, errMessege, http.StatusBadRequest)
	}

	pkColumn, err := explorer.getPKColumn(tableName)
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}

	columns := make([]string, 0, len(updates))
	placeholders := make([]string, 0, len(updates))
	params := make([]interface{}, 0, len(updates))

	for col, val := range updates {
		if pkColumn == col {
			continue
		}
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

		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}

	lastId, err := result.LastInsertId()
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}

	response := CT{
		"response": CT{
			pkColumn: lastId,
		},
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
	}
	return
}

func (explorer *dbExplorer) handleDeleteById(w http.ResponseWriter, r *http.Request) {
	err := explorer.getDbInfo()
	if err != nil {
		_ = writeJSON(w, CT{"error": "getDbInfo error"}, http.StatusInternalServerError)
		return
	}

	tableName := explorer.pathParts[1]

	if explorer.checkTable(tableName) == "" {
		_ = writeJSON(w, CT{"error": "unknown table"}, http.StatusNotFound)
		return
	}
	id, err := explorer.parseId()
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusBadRequest)
		return
	}

	sqlQuery := fmt.Sprintf(`
        DELETE FROM %s
        WHERE id = ?;`, tableName)

	result, err := explorer.db.Exec(sqlQuery, id)
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
		return
	}

	response := CT{
		"response": CT{
			"deleted": rowsAffected,
		},
	}

	if err := writeJSON(w, response, http.StatusOK); err != nil {
		_ = writeJSON(w, CT{"error": ""}, http.StatusInternalServerError)
	}
	return
}
