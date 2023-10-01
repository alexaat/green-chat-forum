package main

import "fmt"

//      _________messages__________________________________________
//     |  id       |  from_id  |  to_id    |  content  |  date     |
//     |  INTEGER  |  INTEGER  |  INTEGER  |  TEXT     |  INTEGER  |

func crerateMessagesTable() error {
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS messages(id INTEGER PRIMARY KEY, from_id INTEGER NOT NULL, to_id INTEGER NOT NULL, content TEXT NOT NULL, date INTEGER NOT NULL)")
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

func insertMessage(message Message) error {
	statement, err := db.Prepare("INSERT INTO messages (from_id, to_id, content, date) VALUES(?,?,?,?)")
	if err != nil {
		return err
	}
	defer statement.Close()

	_, err = statement.Exec(message.FromId, message.ToId, message.Content, message.Date)
	if err != nil {
		return err
	}
	return nil
}

func getChat(from_id int, to_id int, page int) (*[]Message, error) {

	offset := (page - 1) * 10

	query := fmt.Sprintf(
		`
	SELECT
	messages.id, from_id, users.nick_name, to_id, content, date
	FROM messages
	INNER JOIN users ON users.id = from_id
	WHERE from_id = ? AND to_id = ?
	UNION
	SELECT
	messages.id, from_id, users.nick_name, to_id, content, date
	FROM messages
	INNER JOIN users ON users.id = from_id
	WHERE from_id = ? AND to_id = ?	
	ORDER BY date DESC

	LIMIT 10 OFFSET %v 
	`, offset)

	rows, err := db.Query(query, from_id, to_id, to_id, from_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []Message{}

	for rows.Next() {
		var message Message
		err = rows.Scan(&(message.Id), &(message.FromId), &(message.FromNickName), &(message.ToId), &(message.Content), &(message.Date))
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return &messages, nil
}

func getChatMates(id int) ([]*User, error) {

	fmt.Println("id = ", id)

	//Inbox
	// query :=
	// 	`
	// SELECT MAX(date), from_id
	// FROM messages
	// WHERE to_id = ?
	// GROUP BY from_id
	// `

	// //Outbox
	// query =
	// 	`
	// SELECT MAX(date), to_id
	// FROM messages
	// WHERE from_id = ?
	// GROUP BY to_id

	// `

	//Both
	query :=
		`
		SELECT users.id, nick_name
		FROM users
		INNER JOIN 
		(


		SELECT MAX(date), u_id
		FROM
		(
		SELECT MAX(date) AS date, from_id AS u_id
		FROM messages
		WHERE to_id = ?
		GROUP BY u_id
		UNION ALL
		SELECT MAX(date) As date, to_id As u_id
		FROM messages
		WHERE from_id = ?
		GROUP BY u_id		
		)
		GROUP BY u_id
		ORDER BY date DESC


		)
		ON users.id = u_id
	`

	rows, err := db.Query(query, id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}

	for rows.Next() {
		//var userId int
		//var nickName string
		//var maxDate int
		var user User
		//err = rows.Scan(&maxDate, &(user.Id))
		err = rows.Scan(&(user.Id), &(user.NickName))
		//fmt.Println(user.Id, ": ", user.NickName)
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
