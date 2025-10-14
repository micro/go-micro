// Package cliutil contains methods used across all cli commands
// @todo: get rid of os.Exits and use errors instread
package util

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
	merrors "go-micro.dev/v5/errors"
)

type Exec func(*cli.Context, []string) ([]byte, error)

func Print(e Exec) func(*cli.Context) error {
	return func(c *cli.Context) error {
		rsp, err := e(c, c.Args().Slice())
		if err != nil {
			return CliError(err)
		}
		if len(rsp) > 0 {
			fmt.Printf("%s\n", string(rsp))
		}
		return nil
	}
}

// CliError returns a user friendly message from error. If we can't determine a good one returns an error with code 128
func CliError(err error) cli.ExitCoder {
	if err == nil {
		return nil
	}
	// if it's already a cli.ExitCoder we use this
	cerr, ok := err.(cli.ExitCoder)
	if ok {
		return cerr
	}

	// grpc errors
	if mname := regexp.MustCompile(`malformed method name: \\?"(\w+)\\?"`).FindStringSubmatch(err.Error()); len(mname) > 0 {
		return cli.Exit(fmt.Sprintf(`Method name "%s" invalid format. Expecting service.endpoint`, mname[1]), 3)
	}
	if service := regexp.MustCompile(`service ([\w\.]+): route not found`).FindStringSubmatch(err.Error()); len(service) > 0 {
		return cli.Exit(fmt.Sprintf(`Service "%s" not found`, service[1]), 4)
	}
	if service := regexp.MustCompile(`unknown service ([\w\.]+)`).FindStringSubmatch(err.Error()); len(service) > 0 {
		if strings.Contains(service[0], ".") {
			return cli.Exit(fmt.Sprintf(`Service method "%s" not found`, service[1]), 5)
		}
		return cli.Exit(fmt.Sprintf(`Service "%s" not found`, service[1]), 5)
	}
	if address := regexp.MustCompile(`Error while dialing dial tcp.*?([\w]+\.[\w:\.]+): `).FindStringSubmatch(err.Error()); len(address) > 0 {
		return cli.Exit(fmt.Sprintf(`Failed to connect to micro server at %s`, address[1]), 4)
	}

	merr, ok := err.(*merrors.Error)
	if !ok {
		return cli.Exit(err, 128)
	}

	switch merr.Code {
	case 408:
		return cli.Exit("Request timed out", 1)
	case 401:
		// TODO check if not signed in, prompt to sign in
		return cli.Exit("Not authorized to perform this request", 2)
	}

	// fallback to using the detail from the merr
	return cli.Exit(merr.Detail, 127)
}
