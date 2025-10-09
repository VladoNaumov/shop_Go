package handler

//misc.go
import "net/http"

func Health(w http.ResponseWriter, r *http.Request)   { w.WriteHeader(http.StatusOK) }
func NotFound(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }
