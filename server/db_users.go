package main

import (
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//     _________users_______________________________________________________________________________________________
//     |  id      |  first_name  |  last_name  |  age  |  gender  |  nick_name  |  email   |  password | session_id |
//     |  INTEGER |  TEXT        |  TEXT       |  int  |  TEXT    |  TEXT       |  TEXT    |  TEXT     | TEXT       |

func crerateUsersTable() error {
	sql := "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, first_name TEXT, last_name TEXT, age INTEGER, gender TEXT NOT NULL, nick_name TEXT NOT NULL UNIQUE, email TEXT NOT NULL UNIQUE, password TEXT NOT NULL, session_id TEXT)"

	statement, err := db.Prepare(sql)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec()
	if err != nil {
		return err
	}
	return nil
}

func getUsers() ([]*User, error) {
	rows, err := db.Query("SELECT id, nick_name FROM users ORDER BY nick_name COLLATE NOCASE ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		user := User{}
		err = rows.Scan(&(user.Id), &(user.NickName))
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return users, nil
}

func saveUser(user *User) (int64, error) {
	statement, err := db.Prepare("INSERT INTO users (first_name, last_name, age, gender, nick_name, email, password, session_id) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return -1, err
	}
	defer statement.Close()
	result, err := statement.Exec(user.FirstName, user.LastName, user.Age, user.Gender, user.NickName, strings.ToLower(user.Email), user.Password, user.SessionId)
	if err != nil {
		return -1, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return id, nil
}

func printUsers() error {
	rows, err := db.Query("SELECT * FROM users")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		user := User{}
		err = rows.Scan(&(user.Id), &(user.FirstName), &(user.LastName), &(user.Age), &(user.Gender), &(user.NickName), &(user.Email), &(user.Password), &(user.SessionId))
		if err != nil {
			return err
		}
		fmt.Println(user)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	return nil
}

func getUserByEmailOrNickNameAndPassword(user User) (*User, error) {
	u := User{}

	// Get By Email
	rows, err := db.Query("SELECT * FROM users WHERE email = ?", strings.ToLower(strings.TrimSpace(user.NickName)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&(u.Id), &(u.FirstName), &(u.LastName), &(u.Age), &(u.Gender), &(u.NickName), &(u.Email), &(u.Password), &(u.SessionId))
		if err != nil {
			return nil, err
		}
		if compairPasswords(u.Password, user.Password) {
			return &u, nil
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	// Get By Nick Name

	rows, err = db.Query("SELECT * FROM users WHERE nick_name = ?", strings.TrimSpace(user.NickName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&(u.Id), &(u.FirstName), &(u.LastName), &(u.Age), &(u.Gender), &(u.NickName), &(u.Email), &(u.Password), &(u.SessionId))
		if err != nil {
			return nil, err
		}
		if compairPasswords(u.Password, user.Password) {
			return &u, nil
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func updateSessionId(user *User) error {
	statement, err := db.Prepare("UPDATE users SET session_id = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(user.SessionId, user.Id)
	if err != nil {
		return err
	}
	return nil
}

func resetSessionId(sessionId string) error {
	statement, err := db.Prepare("UPDATE users SET session_id = ? WHERE session_id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec("", sessionId)
	if err != nil {
		return err
	}
	return nil
}

func getUserBySessionId(session_id string) (*User, error) {
	if strings.TrimSpace(session_id) == "" {
		return nil, nil
	}
	rows, err := db.Query("SELECT * FROM users WHERE session_id = ? LIMIT 1", session_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var user *User = nil
	for rows.Next() {
		user = &User{}
		err = rows.Scan(&(user.Id), &(user.FirstName), &(user.LastName), &(user.Age), &(user.Gender), &(user.NickName), &(user.Email), &(user.Password), &(user.SessionId))
		if err != nil {
			return nil, err
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return user, nil
}
