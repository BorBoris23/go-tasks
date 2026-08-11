package request

import (
	"fmt"
	"net/http"
	"strconv"

	"go-learning/task4/user"
)

func MakeSearchUsersParams(r *http.Request) (user.SearchUsersParams, error) {
	orderBy, err := strconv.Atoi(r.URL.Query().Get("order_by"))
	if err != nil {
		return user.SearchUsersParams{}, fmt.Errorf("invalid order_by")
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		return user.SearchUsersParams{}, fmt.Errorf("invalid limit")
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		return user.SearchUsersParams{}, fmt.Errorf("invalid offset")
	}

	return user.SearchUsersParams{
		Query:      r.URL.Query().Get("query"),
		OrderField: r.URL.Query().Get("order_field"),
		OrderBy:    orderBy,
		Limit:      limit,
		Offset:     offset,
	}, nil
}
