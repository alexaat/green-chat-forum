package main

import "time"

func getCurrentMilli() int64 {
	return time.Now().UnixNano() / 1000000
}
func formatMilli(date int) string {
	t := time.Unix(int64(date)/1000, 0)
	return t.Format("02-Jan-2006 15:04:05")
}
