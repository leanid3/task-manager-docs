package models

import (
	"errors"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Имитация базы данных
var users = []User{
	{ID: 1, Name: "Alice", Email: "alice@example.com"},
	{ID: 2, Name: "Bob", Email: "bob@example.com"},
	{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
}

func GetAllUsers() []User {
	return users
}

func GetUserByID(id int) (*User, error) {
	for _, user := range users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, errors.New("user not found")
}

func CreateUser(name, email string) *User {
	newID := len(users) + 1
	user := User{
		ID:    newID,
		Name:  name,
		Email: email,
	}
	users = append(users, user)
	return &user
}
