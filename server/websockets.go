package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type Client struct {
	user           *User
	conn           *websocket.Conn
	messageChannel chan []byte
}

// type Online_Users struct {
// 	//Users []string `json:"online_users"`
// 	Users []User `json:"online_users"`
// }

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var clients = make(map[int]*Client)

func addClient(user User, w http.ResponseWriter, r *http.Request) {

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		errorHandler(err)
		return
	}

	client := Client{
		user:           &user,
		conn:           ws,
		messageChannel: make(chan []byte),
	}
	clients[user.Id] = &client

	go writeMessage(user.Id)
	go readMessages(user.Id)

}

func removeClient(id int) {
	if client, ok := clients[id]; ok {
		client.conn.Close()
		delete(clients, id)
		fmt.Printf("Deleted: %v\n", id)
	}
}

func readMessages(id int) {
	defer func() {
		removeClient(id)
		broadcastClientsStatus()
	}()
	for {
		_, message, err := clients[id].conn.ReadMessage()
		if err != nil {
			// Error:  websocket: close 1001 (going away)
			fmt.Println(err, " Connection: ", id)
			return
		}
		fmt.Println("Message ", string(message))
		message = []byte("Using channel. Message: " + string(message))
		for _, conn := range clients {
			conn.messageChannel <- message
		}
	}
}

func writeMessage(id int) {
	client := clients[id]
	defer func() {
		removeClient(id)
	}()
	for {
		select {
		case message, ok := <-client.messageChannel:
			if ok {
				if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
					fmt.Println(err)
					return
				}
			} else {
				if err := client.conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	}
}

func broadcastClientsStatus() {

	for id, client := range clients {

		//Get all users that chatted with current user/id

		chatMates, err := getChatMates(id)

		if err != nil {
			return
		}

		users, err := getUsers()

		if err != nil {
			return
		}

		//Join chatMates and users
		for _, user := range users {
			if !contains(chatMates, *user) && user.Id != id {
				chatMates = append(chatMates, user)
			}
		}

		//Mark on-line/off-line users
		for _, user := range chatMates {
			//Mark on-line/off-line
			setOnLineStatus(user, clients)
		}

		message := `{"online_users":[`
		for _, user := range chatMates {
			message += fmt.Sprintf(`{"id": %v, "nick_name": "%v", "on_line": "%v"},`, user.Id, user.NickName, user.OnLine)
		}
		message = strings.TrimSuffix(message, ",")
		message += `]}`

		//message := `{"online_users":[`
		// for i, c := range clients {

		// 	if i != id {
		// 		m := fmt.Sprintf(`{"id": %v, "nick_name": "%v"},`, c.user.Id, c.user.NickName)
		// 		message += m
		// 	}
		// }

		//message = strings.TrimSuffix(message, ",")

		//message += `]}`
		client.messageChannel <- []byte(message)

	}
}

func notifyClient(id int, message []byte) {
	if client, ok := clients[id]; ok {
		client.messageChannel <- message
	}
}
