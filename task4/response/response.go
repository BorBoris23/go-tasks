package response

import (
	"encoding/json"
	"net/http"

	"go-learning/task4/user"
)

type SearchUser struct {
	Id     int
	Name   string
	Age    int
	About  string
	Gender string
}

func WriteUsers(w http.ResponseWriter, users user.UsersXML) error {
	w.Header().Set("Content-Type", "application/json")

	result := make([]SearchUser, 0, len(users.Users))

	for _, item := range users.Users {
		result = append(result, SearchUser{
			Id:     item.ID,
			Name:   item.FirstName + " " + item.LastName,
			Age:    item.Age,
			About:  item.About,
			Gender: item.Gender,
		})
	}

	return json.NewEncoder(w).Encode(result)
}
