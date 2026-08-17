package main

import (
	"encoding/json"
	"net/http"
)

func writeError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	_ = json.NewEncoder(writer).Encode(map[string]string{
		"error": message,
	})
}

func badRequest(writer http.ResponseWriter, message string) {
	writeError(writer, http.StatusBadRequest, message)
}

func badRequestField(writer http.ResponseWriter, fieldName string) {
	badRequest(writer, "field "+fieldName+" have invalid type")
}

func notFound(writer http.ResponseWriter, message string) {
	writeError(writer, http.StatusNotFound, message)
}

func writeJSON(writer http.ResponseWriter, status int, data interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	_ = json.NewEncoder(writer).Encode(data)
}
