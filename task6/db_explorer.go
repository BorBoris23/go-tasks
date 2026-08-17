package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type DbColumn struct {
	Name    string
	Type    string
	Null    string
	Default interface{}
	Key     string
}

type DbTable struct {
	Name    string
	Columns []DbColumn
}

type TablesResponse struct {
	Response struct {
		Tables []string `json:"tables"`
	} `json:"response"`
}

type ExplorerRequest struct {
	Writer    http.ResponseWriter
	Request   *http.Request
	DB        *sql.DB
	Tables    []DbTable
	Path      string
	Parts     []string
	TableName string
	RecordID  string
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	tables, err := getTables(db)
	if err != nil {
		return nil, err
	}

	handler := func(writer http.ResponseWriter, request *http.Request) {

		req := parseExplorerRequest(writer, request, db, tables)

		switch request.Method {
		case http.MethodGet:
			switch {
			case req.Path == "":
				getTableNamesHandler(writer, tables)
			case len(req.Parts) == 1:
				getRecordsByTableNameHandler(req)
			case len(req.Parts) == 2:
				getRecordByIDHandler(req)
			default:
				http.Error(writer, "bad request", http.StatusBadRequest)
				return
			}

		case http.MethodPut:
			putRecordHandler(req)

		case http.MethodPost:
			postRecordHandler(req)

		case http.MethodDelete:
			deleteRecordHandler(req)

		default:
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		}
	}

	return http.HandlerFunc(handler), nil
}

func deleteRecordHandler(req ExplorerRequest) {

	writer := req.Writer
	tables := req.Tables
	tableName := req.TableName
	db := req.DB
	recordID := req.RecordID

	for _, table := range tables {
		if table.Name != tableName {
			continue
		}

		primaryKey := getPrimaryKey(table.Columns)
		if primaryKey == "" {
			http.Error(writer, "internal server error", http.StatusInternalServerError)
			return
		}

		query := "DELETE FROM " + tableName + " WHERE " + primaryKey + " = ?"

		result, err := db.Exec(query, recordID)
		if err != nil {
			http.Error(writer, "internal server error", http.StatusInternalServerError)
			return
		}

		deleted, err := result.RowsAffected()
		if err != nil {
			http.Error(writer, "internal server error", http.StatusInternalServerError)
			return
		}

		writeJSON(writer, http.StatusOK, map[string]interface{}{
			"response": map[string]interface{}{
				"deleted": deleted,
			},
		})

		return
	}

	notFound(writer, "unknown table")
}

func postRecordHandler(req ExplorerRequest) {

	writer := req.Writer
	tables := req.Tables
	tableName := req.TableName
	db := req.DB
	request := req.Request
	recordID := req.RecordID

	var data map[string]interface{}

	if err := json.NewDecoder(request.Body).Decode(&data); err != nil {
		http.Error(writer, "internal server error", http.StatusInternalServerError)
		return
	}

	for _, table := range tables {
		if table.Name != tableName {
			continue
		}

		primaryKey := getPrimaryKey(table.Columns)
		if primaryKey == "" {
			http.Error(writer, "internal server error", http.StatusInternalServerError)
			return
		}

		var setParts []string
		var values []interface{}

		for _, column := range table.Columns {
			value, exists := data[column.Name]
			if !exists {
				continue
			}

			if column.Name == primaryKey {
				badRequestField(writer, column.Name)
				return
			}

			if value == nil {
				if column.Null == "NO" {
					badRequestField(writer, column.Name)
					return
				}

				setParts = append(setParts, column.Name+" = ?")
				values = append(values, nil)
				continue
			}

			switch {
			case strings.HasPrefix(column.Type, "int"):
				if _, ok := value.(float64); !ok {
					badRequestField(writer, column.Name)
					return
				}

			case strings.HasPrefix(column.Type, "varchar"),
				strings.HasPrefix(column.Type, "text"):
				if _, ok := value.(string); !ok {
					badRequestField(writer, column.Name)
					return
				}
			}

			setParts = append(setParts, column.Name+" = ?")
			values = append(values, value)
		}

		if len(setParts) == 0 {
			badRequest(writer, "no fields to update")
			return
		}

		query := "UPDATE " + tableName +
			" SET " + strings.Join(setParts, ", ") +
			" WHERE " + primaryKey + " = ?"

		values = append(values, recordID)

		result, err := db.Exec(query, values...)
		if err != nil {
			http.Error(writer, "internal server error", http.StatusInternalServerError)
			return
		}

		updated, err := result.RowsAffected()
		if err != nil {
			http.Error(writer, "internal server error", http.StatusInternalServerError)
			return
		}

		if updated == 0 {
			notFound(writer, "record not found")
			return
		}

		writeJSON(writer, http.StatusOK, map[string]interface{}{
			"response": map[string]interface{}{
				"updated": updated,
			},
		})
		return
	}

	notFound(writer, "unknown table")
}

func putRecordHandler(req ExplorerRequest) {

	writer := req.Writer
	tables := req.Tables
	tableName := req.TableName
	db := req.DB
	request := req.Request

	var data map[string]interface{}

	if err := json.NewDecoder(request.Body).Decode(&data); err != nil {
		http.Error(writer, "internal server error", http.StatusInternalServerError)
		return
	}

	for _, table := range tables {
		if table.Name != tableName {
			continue
		}

		var columnNames []string
		var values []interface{}

		for _, column := range table.Columns {
			if column.Key == "PRI" {
				continue
			}

			value, exists := data[column.Name]

			if !exists {
				if column.Null == "YES" {
					value = nil
				} else if strings.HasPrefix(column.Type, "int") {
					value = 0
				} else {
					value = ""
				}
			}

			if value != nil {
				switch {
				case strings.HasPrefix(column.Type, "int"):
					if _, ok := value.(float64); !ok {
						badRequestField(writer, column.Name)
						return
					}

				case strings.HasPrefix(column.Type, "varchar"),
					strings.HasPrefix(column.Type, "text"):
					if _, ok := value.(string); !ok {
						badRequestField(writer, column.Name)
						return
					}
				}
			}

			columnNames = append(columnNames, column.Name)
			values = append(values, value)
		}

		placeholders := make([]string, len(columnNames))

		for i := range placeholders {
			placeholders[i] = "?"
		}

		query := "INSERT INTO " + tableName +
			" (" + strings.Join(columnNames, ", ") + ")" +
			" VALUES (" + strings.Join(placeholders, ", ") + ")"

		result, err := db.Exec(query, values...)
		if err != nil {
			http.Error(writer, "internal server error", http.StatusInternalServerError)
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			http.Error(writer, "internal server error", http.StatusInternalServerError)
			return
		}

		primaryKey := ""

		for _, column := range table.Columns {
			if column.Key == "PRI" {
				primaryKey = column.Name
				break
			}
		}

		if primaryKey == "" {
			http.Error(writer, "internal server error", http.StatusInternalServerError)
			return
		}

		writeJSON(writer, http.StatusOK, map[string]interface{}{
			"response": map[string]interface{}{
				primaryKey: id,
			},
		})
		return
	}

	notFound(writer, "unknown table")
}

func getRecordByIDHandler(req ExplorerRequest) {

	writer := req.Writer
	tables := req.Tables
	tableName := req.TableName
	recordID := req.RecordID
	db := req.DB

	for _, table := range tables {
		if table.Name == tableName {
			columns, err := getTableColumns(db, tableName)
			if err != nil {
				http.Error(writer, "internal server error", http.StatusInternalServerError)
				return
			}

			primaryKey := getPrimaryKey(columns)

			if primaryKey == "" {
				http.Error(writer, "internal server error", http.StatusInternalServerError)
				return
			}

			rows, err := db.Query("SELECT * FROM "+tableName+" WHERE "+primaryKey+" = ?", recordID)
			if err != nil {
				http.Error(writer, "internal server error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			records, err := getRecords(rows, columns)
			if err != nil {
				http.Error(writer, "internal server error", http.StatusInternalServerError)
				return
			}

			if len(records) == 0 {
				notFound(writer, "record not found")
				return
			}

			response := map[string]interface{}{
				"response": map[string]interface{}{
					"record": records[0],
				},
			}

			writeJSON(writer, http.StatusOK, response)

			return
		}
	}
}

func getPrimaryKey(columns []DbColumn) string {
	for _, column := range columns {
		if column.Key == "PRI" {
			return column.Name
		}
		fmt.Printf("column: %+v\n", column)
	}

	return ""
}

func getTables(db *sql.DB) ([]DbTable, error) {
	tables, err := getTableNames(db)
	if err != nil {
		return nil, err
	}

	for i := range tables {
		columns, err := getTableColumns(db, tables[i].Name)
		if err != nil {
			return nil, err
		}

		tables[i].Columns = columns
	}

	return tables, nil
}

func getTableColumns(db *sql.DB, tableName string) ([]DbColumn, error) {
	rows, err := db.Query("SHOW FULL COLUMNS FROM " + tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	var columns []DbColumn

	for rows.Next() {
		values := make([]interface{}, len(columnTypes))
		scanArgs := make([]interface{}, len(columnTypes))

		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		var nullValue string
		if values[3] != nil {
			nullValue = string(values[3].([]byte))
		}

		var keyValue string
		if values[4] != nil {
			keyValue = string(values[4].([]byte))
		}

		column := DbColumn{
			Name:    string(values[0].([]byte)),
			Type:    string(values[1].([]byte)),
			Null:    nullValue,
			Default: values[5],
			Key:     keyValue,
		}

		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

func getTableNames(db *sql.DB) ([]DbTable, error) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []DbTable

	for rows.Next() {
		var table DbTable

		err := rows.Scan(&table.Name)
		if err != nil {
			return nil, err
		}

		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

func getRecordsByTableNameHandler(req ExplorerRequest) {

	writer := req.Writer
	tables := req.Tables
	tableName := req.TableName
	db := req.DB
	request := req.Request

	limit, offset := getLimitAndOffset(request)

	for _, table := range tables {
		if table.Name == tableName {
			columns, err := getTableColumns(db, tableName)
			if err != nil {
				http.Error(writer, "internal server error", http.StatusInternalServerError)
				return
			}

			rows, err := db.Query("SELECT * FROM " + tableName)
			if err != nil {
				http.Error(writer, "internal server error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			records, err := getRecords(rows, columns)
			if err != nil {
				http.Error(writer, "internal server error", http.StatusInternalServerError)
				return
			}

			records = applyOffset(records, offset)
			records = applyLimit(records, limit)

			response := map[string]interface{}{
				"response": map[string]interface{}{
					"records": records,
				},
			}

			writeJSON(writer, http.StatusOK, response)

			return
		}
	}

	notFound(writer, "unknown table")
}

func getLimitAndOffset(request *http.Request) (int, int) {
	limit, err := strconv.Atoi(request.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}

	offset, err := strconv.Atoi(request.URL.Query().Get("offset"))
	if err != nil {
		offset = 0
	}

	return limit, offset
}

func applyOffset(records []map[string]interface{}, offset int) []map[string]interface{} {
	if offset >= len(records) {
		return []map[string]interface{}{}
	}

	return records[offset:]
}

func applyLimit(records []map[string]interface{}, limit int) []map[string]interface{} {
	if limit <= 0 || limit >= len(records) {
		return records
	}

	return records[:limit]
}

func getRecords(rows *sql.Rows, columns []DbColumn) ([]map[string]interface{}, error) {
	var records []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))

		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		record := make(map[string]interface{})

		for i, column := range columns {
			value := values[i]

			if bytes, ok := value.([]byte); ok {
				if strings.HasPrefix(column.Type, "int") {
					var number int
					if _, err := fmt.Sscan(string(bytes), &number); err != nil {
						return nil, err
					}
					record[column.Name] = number
				} else {
					record[column.Name] = string(bytes)
				}
			} else {
				record[column.Name] = value
			}
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func getTableNamesHandler(writer http.ResponseWriter, tables []DbTable) {
	tableNames := make([]string, 0, len(tables))

	for _, table := range tables {
		tableNames = append(tableNames, table.Name)
	}

	response := TablesResponse{}
	response.Response.Tables = tableNames

	writeJSON(writer, http.StatusOK, response)
}
