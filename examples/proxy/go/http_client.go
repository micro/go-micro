package main

import (
	"fmt"
	"net/url"
)

func main() {
	rsp, err := httpCall("/greeter", url.Values{"name": []string{"John"}})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rsp)
}
