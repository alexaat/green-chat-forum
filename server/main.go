package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var db *sql.DB

func main() {
	dbLocal, err := sql.Open("sqlite3", "./forum.db")
	if err != nil {
		log.Fatal(err)
	}
	db = dbLocal
	defer db.Close()
	createTables()
	// err = printUsers()
	if err != nil {
		fmt.Println(err)
	}

	http.Handle("/", http.FileServer(http.Dir("../")))
	http.HandleFunc("/home", homeHandler)
	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/signin", signinHandler)
	http.HandleFunc("/signout", signoutHandler)
	http.HandleFunc("/newpost", newpostHandler)
	http.HandleFunc("/message", messageHandler)
	http.HandleFunc("/messages", messagesHandler)
	http.HandleFunc("/comments", commentsHandler)
	http.HandleFunc("/ws/", websocketHandler)
	fmt.Println("Server running at port 8080")
	http.ListenAndServe(":8080", nil)

}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	session_id := strings.TrimPrefix(r.URL.Path, "/ws/")

	user, err := getUserBySessionId(session_id)
	if err != nil {
		return
	}

	if user == nil {
		return
	}
	addClient(*user, w, r)
	broadcastClientsStatus()
}

func homeHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{Payload: nil, Error: nil}

	if r.Method == "GET" {

		keys, ok := r.URL.Query()["session_id"]
		if !ok || len(keys[0]) < 1 {
			resp.Error = &Error{Type: MISSING_PARAM, Message: "Error: missing request parameter: session_id"}
			json.NewEncoder(w).Encode(resp)
			return
		}
		session_id := keys[0]

		user, err := getUserBySessionId(session_id)
		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if user != nil {
			posts, err := getPosts(user)
			if err != nil {
				resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
				json.NewEncoder(w).Encode(resp)
				return
			}

			removeUserInfo(user)
			data := Data{Posts: posts, User: user}
			resp.Payload = data

		}
	} else {
		resp.Error = &Error{Type: WRONG_METHOD, Message: "Error: wrong http method"}
	}

	json.NewEncoder(w).Encode(resp)
}

func signinHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{Payload: nil, Error: nil}

	user_name := r.FormValue("user_name")
	password := r.FormValue("password")

	user, e := getUserByEmailOrNickNameAndPassword(User{NickName: user_name, Password: password})

	if e != nil {
		resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: unable access database: %v", e)}
	} else {
		if user == nil {
			resp.Error = &Error{Type: NO_USER_FOUND, Message: "Error: no such user"}
		} else {

			user.SessionId = generateSessionId()
			err := updateSessionId(user)
			if err != nil {
				resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: unable access database: %v", err)}
				json.NewEncoder(w).Encode(resp)
				return
			}

			posts, err := getPosts(user)
			if err != nil {
				resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: unable access database: %v", err)}
				json.NewEncoder(w).Encode(resp)
				return
			}

			removeUserInfo(user)
			data := Data{Posts: posts, User: user}
			resp.Payload = data
		}
	}

	json.NewEncoder(w).Encode(resp)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{Payload: nil, Error: nil}

	data := Data{User: &User{}, Posts: nil}

	data.User.FirstName = strings.TrimSpace(r.FormValue("first_name"))
	data.User.LastName = strings.TrimSpace(r.FormValue("last_name"))
	data.User.NickName = strings.TrimSpace(r.FormValue("nick_name"))
	data.User.Email = strings.TrimSpace(r.FormValue("email"))
	data.User.Gender = strings.TrimSpace(r.FormValue("gender"))
	data.User.Password = r.FormValue("password")
	data.User.Password2 = r.FormValue("password2")

	age_str := strings.TrimSpace(r.FormValue("age"))
	resp.Error = validateInput(data.User, age_str)

	if resp.Error == nil {
		// Try to insert User
		data.User.SessionId = generateSessionId()
		data.User.Password = encrypt(data.User.Password)
		data.User.Password2 = ""
		id, err := saveUser(data.User)
		if err != nil {
			if strings.HasPrefix(err.Error(), "UNIQUE constraint failed: users.nick_name") {
				resp.Error = &Error{Type: INVALID_NICK_NAME, Message: "Error: nick name is already in use"}
				resp.Payload = nil
			} else if strings.HasPrefix(err.Error(), "UNIQUE constraint failed: users.email") {
				resp.Error = &Error{Type: INVALID_EMAIL, Message: "Error: email is already in use"}
				resp.Payload = nil
			} else {
				resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
				resp.Payload = nil
			}
		} else {
			data.User.Id = int(id)
		}

	}
	if resp.Error == nil {

		posts, err := getPosts(data.User)
		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: unable access database: %v", err)}
			resp.Payload = nil
			json.NewEncoder(w).Encode(resp)
			return
		}
		data.Posts = posts
		removeUserInfo(data.User)
		resp.Payload = data
	}
	fmt.Println(resp)
	json.NewEncoder(w).Encode(resp)
}

func signoutHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{Payload: nil, Error: nil}

	session_id := r.FormValue("session_id")

	u, err := getUserBySessionId(session_id)
	if err != nil {
		resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(resp)
		return
	}

	err = resetSessionId(session_id)

	if err != nil {
		resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(resp)
		return
	}

	removeClient(u.Id)
	broadcastClientsStatus()
	json.NewEncoder(w).Encode(resp)
}

func newpostHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{Payload: nil, Error: nil}

	if r.Method == "GET" {
		keys, ok := r.URL.Query()["session_id"]
		if !ok || len(keys[0]) < 1 {
			resp.Error = &Error{Type: MISSING_PARAM, Message: "Error: missing request parameter: session_id"}
			json.NewEncoder(w).Encode(resp)
			return
		}
		session_id := keys[0]

		//Verify User
		user, err := getUserBySessionId(session_id)
		if user == nil {
			resp.Error = &Error{Type: NO_USER_FOUND, Message: fmt.Sprintf("Error: unable to authorize user: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		removeUserInfo(user)

		//Get Categories
		categories, err := getCategories()

		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		j, err := json.Marshal(categories)

		if err != nil {
			resp.Error = &Error{Type: ERROR_PARSING_DATA, Message: fmt.Sprintf("Error: unable to parse data %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		npop := NewPostPageObject{}
		npop.User = user
		npop.Categories = string(j)

		resp.Payload = npop

	} else if r.Method == "POST" {
		session_id := r.FormValue("session_id")
		content := strings.TrimSpace(r.FormValue("content"))
		categories := r.FormValue("categories")

		//0. Validate content
		if len(content) == 0 {
			resp.Error = &Error{Type: INVALID_INPUT, Message: "Empty post is not allowed"}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if len(content) > 10000 {
			resp.Error = &Error{Type: INVALID_INPUT, Message: "Post is too large"}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// 1.Verify session_id
		user, err := getUserBySessionId(session_id)
		if user == nil {
			resp.Error = &Error{Type: NO_USER_FOUND, Message: fmt.Sprintf("Error: unable to authorize user: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		removeUserInfo(user)

		// 2. Insert Post
		var arr []string
		err = json.Unmarshal([]byte(categories), &arr)

		if err != nil {
			return
		}

		post := Post{
			UserId:     user.Id,
			Content:    content,
			Categories: arr,
		}
		err = insertPost(user, &post)

		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}
	} else {
		resp.Error = &Error{Type: WRONG_METHOD, Message: "Error: wrong http method"}
	}

	json.NewEncoder(w).Encode(resp)
}

func messageHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{Payload: nil, Error: nil}

	if r.Method != "POST" {
		resp.Error = &Error{Type: WRONG_METHOD, Message: fmt.Sprintf("Error: %v", "Wrong method used")}
		json.NewEncoder(w).Encode(resp)
		return
	}

	session_id := r.FormValue("session_id")
	to_id := r.FormValue("to_id")
	message := strings.TrimSpace(r.FormValue("message"))

	//Verify input
	if len(message) == 0 {
		resp.Error = &Error{Type: INVALID_INPUT, Message: "Empty message is not allowed"}
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(message) > 1000 {
		resp.Error = &Error{Type: INVALID_INPUT, Message: "Message is too large"}
		json.NewEncoder(w).Encode(resp)
		return
	}

	user, err := getUserBySessionId(session_id)
	if err != nil {
		resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(resp)
		return
	}
	if user == nil {
		json.NewEncoder(w).Encode(resp)
		return
	}

	to_id_int, err := strconv.Atoi(to_id)
	if err != nil {
		resp.Error = &Error{Type: ERROR_PARSING_DATA, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(resp)
		return
	}

	removeUserInfo(user)

	resp.Payload = user

	m := Message{
		FromId:       user.Id,
		FromNickName: user.NickName,
		ToId:         to_id_int,
		Content:      message,
		Date:         getCurrentMilli(),
	}

	err = insertMessage(m)
	if err != nil {
		resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(resp)
		return
	}

	mw := MessageWrapper{m}

	b, err := json.Marshal(mw)

	if err != nil {
		resp.Error = &Error{Type: ERROR_PARSING_DATA, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(resp)
		return
	}

	//Notify both sender and receiver
	notifyClient(m.FromId, b)
	notifyClient(m.ToId, b)

	json.NewEncoder(w).Encode(resp)
}

func commentsHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{Payload: nil, Error: nil}

	if r.Method == "GET" {
		//1.Get session_id and post_id

		keys, ok := r.URL.Query()["session_id"]
		if !ok || len(keys[0]) < 1 {
			resp.Error = &Error{Type: MISSING_PARAM, Message: "Error: missing request parameter: session_id"}
			json.NewEncoder(w).Encode(resp)
			return
		}
		session_id := keys[0]

		keys, ok = r.URL.Query()["post_id"]
		if !ok || len(keys[0]) < 1 {
			resp.Error = &Error{Type: MISSING_PARAM, Message: "Error: missing request parameter: post_id"}
			json.NewEncoder(w).Encode(resp)
			return
		}
		post_id := keys[0]

		// 2.Verify session_id
		user, err := getUserBySessionId(session_id)
		if user == nil {
			resp.Error = &Error{Type: NO_USER_FOUND, Message: fmt.Sprintf("Error: unable to authorize user: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		removeUserInfo(user)

		//3. Get Post by post_id
		postId, err := strconv.Atoi(post_id)
		if err != nil {
			resp.Error = &Error{Type: ERROR_PARSING_DATA, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}
		post, err := getPost(postId)
		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		//4. Get Comments
		comments, err := getComments(postId)
		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		cpo := CommentsPageObject{}
		cpo.User = user
		cpo.Post = post
		cpo.Comments = comments
		resp.Payload = cpo

	} else if r.Method == "POST" {
		session_id := r.FormValue("session_id")
		post_id := r.FormValue("post_id")
		comment := strings.TrimSpace(r.FormValue("comment"))

		//Verify comment
		if len(comment) == 0 {
			resp.Error = &Error{Type: INVALID_INPUT, Message: "Empty comment is not allowed"}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if len(comment) > 10000 {
			resp.Error = &Error{Type: INVALID_INPUT, Message: "Comment is too large"}
			json.NewEncoder(w).Encode(resp)
			return
		}

		user, err := getUserBySessionId(session_id)
		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if user == nil {
			resp.Error = &Error{Type: NO_USER_FOUND, Message: fmt.Sprintf("Error: unable to authorize user: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		postId, err := strconv.Atoi(post_id)

		if err != nil {
			resp.Error = &Error{Type: ERROR_PARSING_DATA, Message: fmt.Sprintf("Error: unable to parse data %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

		err = saveComment(user.Id, postId, comment)

		if err != nil {
			resp.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
			json.NewEncoder(w).Encode(resp)
			return
		}

	} else {
		resp.Error = &Error{Type: WRONG_METHOD, Message: "Error: wrong http method"}
	}
	json.NewEncoder(w).Encode(resp)
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	chat := Chat{UserId: -1, ChatMateId: -1, Messages: nil, Error: nil}

	if r.Method != "POST" {
		chat.Error = &Error{Type: WRONG_METHOD, Message: fmt.Sprintf("Error: %v", "Wrong method used")}
		json.NewEncoder(w).Encode(chat)
		return
	}

	session_id := r.FormValue("session_id")

	page, err := strconv.Atoi(r.FormValue("page"))
	if err != nil {
		chat.Error = &Error{Type: ERROR_PARSING_DATA, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(chat)
		return
	}
	if page <= 0 {
		page = 1
	}

	chat_mate_id, err := strconv.Atoi(r.FormValue("chat_mate_id"))
	if err != nil {
		chat.Error = &Error{Type: ERROR_PARSING_DATA, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(chat)
		return
	}

	chat.ChatMateId = chat_mate_id

	user, err := getUserBySessionId(session_id)
	if err != nil {
		chat.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(chat)
		return
	}

	chat.UserId = user.Id

	messages, err := getChat(user.Id, chat_mate_id, page)

	if err != nil {
		chat.Error = &Error{Type: ERROR_ACCESSING_DATABASE, Message: fmt.Sprintf("Error: %v", err)}
		json.NewEncoder(w).Encode(chat)
		return
	}

	chat.Messages = messages

	json.NewEncoder(w).Encode(chat)
}

func errorHandler(err error) {
	fmt.Println("Error: ", err)
}

func Cors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func validateInput(user *User, age_str string) *Error {
	if len(user.FirstName) < 2 || len(user.FirstName) > 50 {
		return &Error{Type: INVALID_FIRST_NAME, Message: "Error: first name should be between 2 and 50 characters long"}
	}
	if len(user.LastName) < 2 || len(user.LastName) > 50 {
		return &Error{Type: INVALID_LAST_NAME, Message: "Error: last name should be between 2 and 50 characters long"}
	}

	i, err := strconv.Atoi(age_str)
	if err != nil {
		return &Error{Type: INVALID_AGE, Message: "Error: invalid age"}
	}
	user.Age = i

	if user.Age < 0 || user.Age > 200 {
		return &Error{Type: INVALID_AGE, Message: "Error: invalid age"}
	}

	//Validate gender
	if !(user.Gender == "Male" || user.Gender == "Female" || user.Gender == "Other" || user.Gender == "Prefer Not To Say") {
		return &Error{Type: INVALID_GENDER, Message: "Error: invalid gender option"}
	}

	if len(user.NickName) < 2 || len(user.NickName) > 50 {
		return &Error{Type: INVALID_NICK_NAME, Message: "Error: nick name should be between 2 and 50 characters long"}
	}

	// Check for valid email
	reg := `^[^@\s]+@[^@\s]+.[^@\s]$`
	match, err := regexp.MatchString(reg, user.Email)
	if err != nil || !match {
		return &Error{Type: INVALID_EMAIL, Message: "Error: invalid email"}
	}

	// Validate password
	if len(user.Password) < 6 || len(user.Password) > 50 {
		return &Error{Type: INVALID_PASSWORD, Message: "Error: password should be between 6 and 50 characters long"}
	}
	// Validate passwords
	if user.Password != user.Password2 {
		return &Error{Type: INVALID_PASSWORD_2, Message: "Error: passwords don't match"}
	}

	return nil
}

func createTables() {

	err := crerateUsersTable()
	if err != nil {
		log.Fatal(err)
	}
	err = creratePostsTable()
	if err != nil {
		log.Fatal(err)
	}
	err = crerateMessagesTable()
	if err != nil {
		log.Fatal(err)
	}
	err = crerateCategoriesTable()
	if err != nil {
		log.Fatal(err)
	}
	_ = insertCategories([]string{"gereen apple", "cucumber", "kivi", "green grapes", "avocado", "broccoli", "spinach"})

	err = crerateCommentsTable()
	if err != nil {
		log.Fatal(err)
	}
}

func removeUserInfo(user *User) {
	user.FirstName = ""
	user.LastName = ""
	user.Age = 0
	user.Password = ""
	user.Password2 = ""
	user.Email = ""
	user.Gender = ""
}
