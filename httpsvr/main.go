package main

import (
	"fmt"
	"net/http"
)

func Hello(resp http.ResponseWriter,req *http.Request)  {
	resp.Write([]byte("hello"))
}

func main()  {

	http.HandleFunc("/",Hello)
	e:= http.ListenAndServe(":21212",nil)
	if e!=nil{
		fmt.Println(e)
	}
}