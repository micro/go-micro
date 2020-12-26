// +build main3

package main

import (
	"fmt"
)

func main() {
	rsp, err := rpcCall("/greeter/say/hello", map[string]interface{}{"name": "John"})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rsp)
}
