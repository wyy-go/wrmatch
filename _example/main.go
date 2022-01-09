package main

import (
	"log"
	"net/http"

	"github.com/wyy-go/wrmatch"
)

func main() {
	router := wrmatch.New()
	router.GET("/", "/")
	router.GET("/hello/:name", "Hello")
	router.Add(http.MethodGet,"/test","match")

	v, _, matched := router.Match(http.MethodGet, "/")
	if matched {
		log.Println(v)
	}
	v, ps, matched := router.Match(http.MethodGet, "/hello/myname")
	if matched {
		log.Println(v)
		log.Println(ps.Param("name"))
	}

	v, _, matched = router.Match(http.MethodGet, "/test")
	if matched {
		log.Println(v)
	}
}
