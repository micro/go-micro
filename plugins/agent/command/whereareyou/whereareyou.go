package whereareyou

/*
	Whereareyou command for the Micro Bot

	usage: where are you?
*/

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"github.com/micro/go-micro/v2/agent/command"
)

func init() {
	command.Commands[`^where are you\??`] = WhereAreYou()
}

func WhereAreYou() command.Command {
	usage := "where are you?"
	desc := "Returns the location of the bot"

	getIp := func() string {
		ifaces, _ := net.Interfaces()

		for _, i := range ifaces {
			addrs, _ := i.Addrs()

			for _, addr := range addrs {
				if ip, ok := addr.(*net.IPNet); ok && ip.IP.IsLoopback() {
					continue
				}

				switch v := addr.(type) {
				case *net.IPNet:
					return v.IP.String()
				case *net.IPAddr:
					return v.IP.String()
				}
			}
		}

		return "127.0.0.1"
	}

	return command.NewCommand("whereareyou", usage, desc, func(args ...string) ([]byte, error) {
		rsp, err := http.Get("http://myexternalip.com/raw")
		if err != nil {
			return nil, err
		}
		defer rsp.Body.Close()

		b, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			return nil, err
		}

		host, _ := os.Hostname()
		exIp := string(b)
		inIp := getIp()

		val := fmt.Sprintf("hostname: %s\ninternal ip: %s\nexternal ip: %s", host, inIp, exIp)
		return []byte(val), nil
	})
}
