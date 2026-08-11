package user

import (
	"encoding/xml"
	"errors"
	"os"
	"sort"
	"strings"
)

type SearchUsersParams struct {
	Query      string
	OrderField string
	OrderBy    int
	Limit      int
	Offset     int
}

type UserRecord struct {
	ID        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	About     string `xml:"about"`
	Age       int    `xml:"age"`
	Gender    string `xml:"gender"`
}

type UsersXML struct {
	Users []UserRecord `xml:"row"`
}

const (
	OrderByAsc  = -1
	OrderByAsIs = 0
	OrderByDesc = 1

	OrderFieldID   = "Id"
	OrderFieldAge  = "Age"
	OrderFieldName = "Name"
)

func GetFilteredUsers(
	path string,
	params SearchUsersParams,
) (UsersXML, error) {

	if err := validateOrderField(params.OrderField); err != nil {
		return UsersXML{}, err
	}

	users := makeUserRecords(path)

	filteredUsers := filterUsers(params.Query, users)

	sortedUsers := sortUsers(
		filteredUsers,
		params.OrderField,
		params.OrderBy,
	)

	return paginateUsers(
		sortedUsers,
		params.Offset,
		params.Limit,
	), nil
}

func filterUsers(query string, usersXML UsersXML) UsersXML {
	filteredUsers := UsersXML{
		Users: []UserRecord{},
	}

	for _, user := range usersXML.Users {
		name := user.FirstName + " " + user.LastName

		if query == "" || strings.Contains(name, query) || strings.Contains(user.About, query) {

			filteredUsers.Users = append(filteredUsers.Users, user)
		}
	}
	return filteredUsers
}

func sortUsers(
	users UsersXML,
	orderField string,
	orderBy int,
) UsersXML {
	if orderBy == OrderByAsIs {
		return users
	}

	sort.Slice(users.Users, func(i, j int) bool {
		var less bool

		switch orderField {
		case "", OrderFieldName:
			nameI := users.Users[i].FirstName + " " + users.Users[i].LastName
			nameJ := users.Users[j].FirstName + " " + users.Users[j].LastName

			less = nameI < nameJ

		case OrderFieldID:
			less = users.Users[i].ID < users.Users[j].ID

		case OrderFieldAge:
			less = users.Users[i].Age < users.Users[j].Age
		}

		if orderBy == OrderByDesc {
			return !less
		}

		return less
	})

	return users
}

func validateOrderField(orderField string) error {
	switch orderField {
	case "", OrderFieldName, OrderFieldID, OrderFieldAge:
		return nil
	default:
		return errors.New("ErrorBadOrderField")
	}
}

func makeUserRecords(path string) UsersXML {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var usersSlice UsersXML

	err = xml.Unmarshal(data, &usersSlice)
	if err != nil {
		panic(err)
	}

	return usersSlice
}

func paginateUsers(users UsersXML, offset int, limit int) UsersXML {
	if offset >= len(users.Users) {
		return UsersXML{}
	}

	end := offset + limit

	if end > len(users.Users) {
		end = len(users.Users)
	}

	return UsersXML{
		Users: users.Users[offset:end],
	}
}
