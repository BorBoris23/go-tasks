package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-learning/task4/request"
	"go-learning/task4/response"
	"go-learning/task4/user"
)

func SearchServer(path string, params user.SearchUsersParams) (user.UsersXML, error) {
	return user.GetFilteredUsers(path, params)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	params, err := request.MakeSearchUsersParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	users, err := SearchServer("dataset.xml", params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := response.WriteUsers(w, users); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func badOrderFieldHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	_ = json.NewEncoder(w).Encode(SearchErrorResponse{
		Error: "ErrorBadOrderField",
	})
}

type timeoutError struct{}

func (timeoutError) Error() string {
	return "timeout"
}

func (timeoutError) Timeout() bool {
	return true
}

func (timeoutError) Temporary() bool {
	return true
}

type timeoutRoundTripper struct{}

func (timeoutRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, timeoutError{}
}

func TestFindUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(searchHandler))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	req := SearchRequest{
		Limit:      10,
		Offset:     0,
		Query:      "",
		OrderField: "Name",
		OrderBy:    OrderByAsc,
	}

	result, err := client.FindUsers(req)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Users) != 10 {
		t.Fatalf("expected 10 users, got %d", len(result.Users))
	}

	if !result.NextPage {
		t.Fatal("expected NextPage to be true")
	}
}

func TestFindUsersPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(searchHandler))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	req := SearchRequest{
		Limit:      5,
		Offset:     10,
		Query:      "",
		OrderField: "Name",
		OrderBy:    OrderByAsc,
	}

	result, err := client.FindUsers(req)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Users) != 5 {
		t.Fatalf("expected 5 users, got %d", len(result.Users))
	}

	if !result.NextPage {
		t.Fatal("expected NextPage to be true")
	}
}

func TestFindUsersQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(searchHandler))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	req := SearchRequest{
		Limit:      10,
		Offset:     0,
		Query:      "excepteur",
		OrderField: "Name",
		OrderBy:    OrderByAsc,
	}

	result, err := client.FindUsers(req)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Users) == 0 {
		t.Fatal("expected users, got empty result")
	}

	for _, user := range result.Users {
		if !strings.Contains(user.About, "excepteur") {
			t.Fatalf("user %d does not contain query in About", user.Id)
		}
	}
}

func TestFindUsersByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(searchHandler))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	req := SearchRequest{
		Limit:      10,
		Offset:     0,
		Query:      "Allison",
		OrderField: "Name",
		OrderBy:    OrderByAsc,
	}

	result, err := client.FindUsers(req)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Users) == 0 {
		t.Fatal("expected users, got empty result")
	}

	for _, user := range result.Users {
		t.Logf("FOUND USER: %+v", user)

		name := user.Name

		if !strings.Contains(name, "Allison") {
			t.Fatalf("user %q does not contain query", name)
		}
	}
}

func TestFindUsersInvalidLimit(t *testing.T) {
	client := SearchClient{
		URL: "http://localhost:8080",
	}

	_, err := client.FindUsers(SearchRequest{
		Limit: -1,
	})

	if err == nil {
		t.Fatal("expected error for negative limit")
	}
}

func TestFindUsersInvalidOffset(t *testing.T) {
	client := SearchClient{
		URL: "http://localhost:8080",
	}

	_, err := client.FindUsers(SearchRequest{
		Limit:  10,
		Offset: -1,
	})

	if err == nil {
		t.Fatal("expected error for negative offset")
	}
}

func TestFindUsersSorting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(searchHandler))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	tests := []struct {
		name       string
		orderField string
		orderBy    int
	}{
		{
			name:       "sort by name ascending",
			orderField: "Name",
			orderBy:    OrderByAsc,
		},
		{
			name:       "sort by name descending",
			orderField: "Name",
			orderBy:    OrderByDesc,
		},
		{
			name:       "sort by id ascending",
			orderField: "Id",
			orderBy:    OrderByAsc,
		},
		{
			name:       "sort by id descending",
			orderField: "Id",
			orderBy:    OrderByDesc,
		},
		{
			name:       "sort by age ascending",
			orderField: "Age",
			orderBy:    OrderByAsc,
		},
		{
			name:       "sort by age descending",
			orderField: "Age",
			orderBy:    OrderByDesc,
		},
		{
			name:       "sort by name by default",
			orderField: "",
			orderBy:    OrderByAsc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.FindUsers(SearchRequest{
				Limit:      10,
				Offset:     0,
				OrderField: tt.orderField,
				OrderBy:    tt.orderBy,
			})
			if err != nil {
				t.Fatal(err)
			}

			if len(result.Users) < 2 {
				t.Fatal("expected at least 2 users")
			}

			for i := 1; i < len(result.Users); i++ {
				prev := result.Users[i-1]
				curr := result.Users[i]

				switch tt.orderField {
				case "", "Name":
					if tt.orderBy == OrderByAsc && prev.Name > curr.Name {
						t.Fatalf("users are not sorted by name ascending: %q > %q", prev.Name, curr.Name)
					}

					if tt.orderBy == OrderByDesc && prev.Name < curr.Name {
						t.Fatalf("users are not sorted by name descending: %q < %q", prev.Name, curr.Name)
					}

				case "Id":
					if tt.orderBy == OrderByAsc && prev.Id > curr.Id {
						t.Fatalf("users are not sorted by id ascending: %d > %d", prev.Id, curr.Id)
					}

					if tt.orderBy == OrderByDesc && prev.Id < curr.Id {
						t.Fatalf("users are not sorted by id descending: %d < %d", prev.Id, curr.Id)
					}

				case "Age":
					if tt.orderBy == OrderByAsc && prev.Age > curr.Age {
						t.Fatalf("users are not sorted by age ascending: %d > %d", prev.Age, curr.Age)
					}

					if tt.orderBy == OrderByDesc && prev.Age < curr.Age {
						t.Fatalf("users are not sorted by age descending: %d < %d", prev.Age, curr.Age)
					}
				}
			}
		})
	}
}

func TestFindUsersOrderByAsIs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(searchHandler))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	result, err := client.FindUsers(SearchRequest{
		Limit:      10,
		Offset:     0,
		Query:      "",
		OrderField: "Name",
		OrderBy:    OrderByAsIs,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Users) == 0 {
		t.Fatal("expected users, got empty result")
	}

	if result.Users[0].Id != 0 {
		t.Fatalf("expected first user ID 0, got %d", result.Users[0].Id)
	}
}

func TestFindUsersLimitMax(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(searchHandler))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	result, err := client.FindUsers(SearchRequest{
		Limit:      100,
		Offset:     0,
		Query:      "",
		OrderField: "Name",
		OrderBy:    OrderByAsc,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Users) != 25 {
		t.Fatalf("expected 25 users, got %d", len(result.Users))
	}
}

func TestFindUsersUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := SearchClient{
		AccessToken: "wrong_token",
		URL:         server.URL,
	}

	_, err := client.FindUsers(SearchRequest{
		Limit:  10,
		Offset: 0,
	})

	if err == nil {
		t.Fatal("expected unauthorized error")
	}

	if err.Error() != "Bad AccessToken" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindUsersInternalServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	_, err := client.FindUsers(SearchRequest{
		Limit:  10,
		Offset: 0,
	})

	if err == nil {
		t.Fatal("expected internal server error")
	}

	if err.Error() != "SearchServer fatal error" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindUsersBadOrderField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(badOrderFieldHandler))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	_, err := client.FindUsers(SearchRequest{
		Limit:      10,
		Offset:     0,
		OrderField: "Foo",
		OrderBy:    OrderByAsc,
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "Foo") {
		t.Fatalf("expected error to contain order field, got: %v", err)
	}
}

func TestFindUsersUnknownBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		_ = json.NewEncoder(w).Encode(SearchErrorResponse{
			Error: "some unknown error",
		})
	}))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	_, err := client.FindUsers(SearchRequest{
		Limit:  10,
		Offset: 0,
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "some unknown error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindUsersInvalidErrorJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	_, err := client.FindUsers(SearchRequest{
		Limit:  10,
		Offset: 0,
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "cant unpack error json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindUsersInvalidResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := SearchClient{
		AccessToken: "test_token",
		URL:         server.URL,
	}

	_, err := client.FindUsers(SearchRequest{
		Limit:  10,
		Offset: 0,
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "cant unpack result json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindUsersConnectionError(t *testing.T) {
	client := SearchClient{
		AccessToken: "test_token",
		URL:         "http://127.0.0.1:1",
	}

	_, err := client.FindUsers(SearchRequest{
		Limit:  10,
		Offset: 0,
	})

	if err == nil {
		t.Fatal("expected connection error")
	}

	if strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected non-timeout error, got: %v", err)
	}
}

func TestFindUsersTimeout(t *testing.T) {
	oldClient := client
	defer func() {
		client = oldClient
	}()

	client = &http.Client{
		Transport: timeoutRoundTripper{},
	}

	searchClient := SearchClient{
		URL: "http://example.com",
	}

	_, err := searchClient.FindUsers(SearchRequest{
		Limit:  10,
		Offset: 0,
	})

	if err == nil {
		t.Fatal("expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}
