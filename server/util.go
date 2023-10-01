package main

func contains(arr []*User, user User) bool {
	for i := 0; i < len(arr); i++ {
		if arr[i].Id == user.Id {
			return true
		}
	}
	return false
}

func setOnLineStatus(user *User, clients map[int]*Client) {
	if _, ok := clients[user.Id]; ok {
		user.OnLine = true
	} else {
		user.OnLine = false
	}
}
