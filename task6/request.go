package main

import (
	"database/sql"
	"net/http"
	"strings"
)

func parseExplorerRequest(
	writer http.ResponseWriter,
	request *http.Request,
	db *sql.DB,
	tables []DbTable,
) ExplorerRequest {
	path := strings.Trim(request.URL.Path, "/")
	parts := strings.Split(path, "/")

	req := ExplorerRequest{
		Writer:  writer,
		Request: request,
		DB:      db,
		Tables:  tables,
		Path:    path,
		Parts:   parts,
	}

	if len(parts) > 0 && parts[0] != "" {
		req.TableName = parts[0]
	}

	if len(parts) > 1 {
		req.RecordID = parts[1]
	}

	return req
}
