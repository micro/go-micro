package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/stretchr/objx"
	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry"
)

// LookupService queries the service for a service with the given alias. If
// no services are found for a given alias, the registry will return nil and
// the error will also be nil. An error is only returned if there was an issue
// listing from the registry.
func LookupService(name string) (*registry.Service, error) {
	// return a lookup in the default domain as a catch all
	return serviceWithName(name)
}

// FormatServiceUsage returns a string containing the service usage.
func FormatServiceUsage(srv *registry.Service, c *cli.Context) string {
	alias := c.Args().First()
	subcommand := c.Args().Get(1)

	commands := make([]string, len(srv.Endpoints))
	endpoints := make([]*registry.Endpoint, len(srv.Endpoints))
	for i, e := range srv.Endpoints {
		// map "Helloworld.Call" to "helloworld.call"
		parts := strings.Split(e.Name, ".")
		for i, part := range parts {
			parts[i] = lowercaseInitial(part)
		}
		name := strings.Join(parts, ".")

		// remove the prefix if it is the service name, e.g. rather than
		// "micro run helloworld helloworld call", it would be
		// "micro run helloworld call".
		name = strings.TrimPrefix(name, alias+".")

		// instead of "micro run helloworld foo.bar", the command should
		// be "micro run helloworld foo bar".
		commands[i] = strings.Replace(name, ".", " ", 1)
		endpoints[i] = e
	}

	result := ""
	if len(subcommand) > 0 && subcommand != "--help" {
		result += fmt.Sprintf("NAME:\n\tmicro %v %v\n\n", alias, subcommand)
		result += fmt.Sprintf("USAGE:\n\tmicro %v %v [flags]\n\n", alias, subcommand)
		result += fmt.Sprintf("FLAGS:\n")

		for i, command := range commands {
			if command == subcommand {
				result += renderFlags(endpoints[i])
			}
		}
	} else {
		// sort the command names alphabetically
		sort.Strings(commands)

		result += fmt.Sprintf("NAME:\n\tmicro %v\n\n", alias)
		result += fmt.Sprintf("VERSION:\n\t%v\n\n", srv.Version)
		result += fmt.Sprintf("USAGE:\n\tmicro %v [command]\n\n", alias)
		result += fmt.Sprintf("COMMANDS:\n\t%v\n", strings.Join(commands, "\n\t"))

	}

	return result
}

func lowercaseInitial(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

func renderFlags(endpoint *registry.Endpoint) string {
	ret := ""
	for _, value := range endpoint.Request.Values {
		ret += renderValue([]string{}, value) + "\n"
	}
	return ret
}

func renderValue(path []string, value *registry.Value) string {
	if len(value.Values) > 0 {
		renders := []string{}
		for _, v := range value.Values {
			renders = append(renders, renderValue(append(path, value.Name), v))
		}
		return strings.Join(renders, "\n")
	}
	return fmt.Sprintf("\t--%v %v", strings.Join(append(path, value.Name), "_"), value.Type)
}

// CallService will call a service using the arguments and flags provided
// in the context. It will print the result or error to stdout. If there
// was an error performing the call, it will be returned.
func CallService(srv *registry.Service, args []string) error {
	// parse the flags and args
	args, flags, err := splitCmdArgs(args)
	if err != nil {
		return err
	}

	// construct the endpoint
	endpoint, err := constructEndpoint(args)
	if err != nil {
		return err
	}

	// ensure the endpoint exists on the service
	var ep *registry.Endpoint
	for _, e := range srv.Endpoints {
		if e.Name == endpoint {
			ep = e
			break
		}
	}
	if ep == nil {
		return fmt.Errorf("Endpoint %v not found for service %v", endpoint, srv.Name)
	}

	// parse the flags
	body, err := FlagsToRequest(flags, ep.Request)
	if err != nil {
		return err
	}

	// create a context for the call based on the cli context
	callCtx := context.TODO()

	// TODO: parse out --header or --metadata

	// construct and execute the request using the json content type
	req := client.DefaultClient.NewRequest(srv.Name, endpoint, body, client.WithContentType("application/json"))
	var rsp json.RawMessage

	if err := client.DefaultClient.Call(callCtx, req, &rsp); err != nil {
		return err
	}

	// format the response
	var out bytes.Buffer
	defer out.Reset()
	if err := json.Indent(&out, rsp, "", "\t"); err != nil {
		return err
	}
	out.Write([]byte("\n"))
	out.WriteTo(os.Stdout)

	return nil
}

// splitCmdArgs takes a cli context and parses out the args and flags, for
// example "micro helloworld --name=foo call apple" would result in "call",
// "apple" as args and {"name":"foo"} as the flags.
func splitCmdArgs(arguments []string) ([]string, map[string][]string, error) {
	args := []string{}
	flags := map[string][]string{}

	prev := ""
	for _, a := range arguments {
		if !strings.HasPrefix(a, "--") {
			if len(prev) == 0 {
				args = append(args, a)
				continue
			}
			_, exists := flags[prev]
			if !exists {
				flags[prev] = []string{}
			}

			flags[prev] = append(flags[prev], a)
			prev = ""
			continue
		}

		// comps would be "foo", "bar" for "--foo=bar"
		comps := strings.Split(strings.TrimPrefix(a, "--"), "=")
		_, exists := flags[comps[0]]
		if !exists {
			flags[comps[0]] = []string{}
		}
		switch len(comps) {
		case 1:
			prev = comps[0]
		case 2:
			flags[comps[0]] = append(flags[comps[0]], comps[1])
		default:
			return nil, nil, fmt.Errorf("Invalid flag: %v. Expected format: --foo=bar", a)
		}
	}

	return args, flags, nil
}

// constructEndpoint takes a slice of args and converts it into a valid endpoint
// such as Helloworld.Call or Foo.Bar, it will return an error if an invalid number
// of arguments were provided
func constructEndpoint(args []string) (string, error) {
	var epComps []string
	switch len(args) {
	case 1:
		epComps = append(args, "call")
	case 2:
		epComps = args
	case 3:
		epComps = args[1:3]
	default:
		return "", fmt.Errorf("Incorrect number of arguments")
	}

	// transform the endpoint components, e.g ["helloworld", "call"] to the
	// endpoint name: "Helloworld.Call".
	return fmt.Sprintf("%v.%v", strings.Title(epComps[0]), strings.Title(epComps[1])), nil
}

// ShouldRenderHelp returns true if the help flag was passed
func ShouldRenderHelp(args []string) bool {
	args, flags, _ := splitCmdArgs(args)

	// only 1 arg e.g micro helloworld
	if len(args) == 1 {
		return true
	}

	for key := range flags {
		if key == "help" {
			return true
		}
	}

	return false
}

// FlagsToRequest parses a set of flags, e.g {name:"Foo", "options_surname","Bar"} and
// converts it into a request body. If the key is not a valid object in the request, an
// error will be returned.
//
// This function constructs []interface{} slices
// as opposed to typed ([]string etc) slices for easier testing
func FlagsToRequest(flags map[string][]string, req *registry.Value) (map[string]interface{}, error) {
	coerceValue := func(valueType string, value []string) (interface{}, error) {
		switch valueType {
		case "bool":
			if len(value) == 0 || len(strings.TrimSpace(value[0])) == 0 {
				return true, nil
			}
			return strconv.ParseBool(value[0])
		case "int32":
			i, err := strconv.Atoi(value[0])
			if err != nil {
				return nil, err
			}
			if i < math.MinInt32 || i > math.MaxInt32 {
				return nil, fmt.Errorf("value out of range for int32: %d", i)
			}
			return int32(i), nil
		case "int64":
			return strconv.ParseInt(value[0], 0, 64)
		case "float64":
			return strconv.ParseFloat(value[0], 64)
		case "[]bool":
			// length is one if it's a `,` separated int slice
			if len(value) == 1 {
				value = strings.Split(value[0], ",")
			}
			ret := []interface{}{}
			for _, v := range value {
				i, err := strconv.ParseBool(v)
				if err != nil {
					return nil, err
				}
				ret = append(ret, i)
			}
			return ret, nil
		case "[]int32":
			// length is one if it's a `,` separated int slice
			if len(value) == 1 {
				value = strings.Split(value[0], ",")
			}
			ret := []interface{}{}
			for _, v := range value {
				i, err := strconv.Atoi(v)
				if err != nil {
					return nil, err
				}
				if i < math.MinInt32 || i > math.MaxInt32 {
					return nil, fmt.Errorf("value out of range for int32: %d", i)
				}
				ret = append(ret, int32(i))
			}
			return ret, nil
		case "[]int64":
			// length is one if it's a `,` separated int slice
			if len(value) == 1 {
				value = strings.Split(value[0], ",")
			}
			ret := []interface{}{}
			for _, v := range value {
				i, err := strconv.ParseInt(v, 0, 64)
				if err != nil {
					return nil, err
				}
				ret = append(ret, i)
			}
			return ret, nil
		case "[]float64":
			// length is one if it's a `,` separated float slice
			if len(value) == 1 {
				value = strings.Split(value[0], ",")
			}
			ret := []interface{}{}
			for _, v := range value {
				i, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, err
				}
				ret = append(ret, i)
			}
			return ret, nil
		case "[]string":
			// length is one it's a `,` separated string slice
			if len(value) == 1 {
				value = strings.Split(value[0], ",")
			}
			ret := []interface{}{}
			for _, v := range value {
				ret = append(ret, v)
			}
			return ret, nil
		case "string":
			return value[0], nil
		case "map[string]string":
			var val map[string]string
			if err := json.Unmarshal([]byte(value[0]), &val); err != nil {
				return value[0], nil
			}
			return val, nil
		default:
			return value, nil
		}
		return nil, nil
	}

	result := objx.MustFromJSON("{}")

	var flagType func(key string, values []*registry.Value, path ...string) (string, bool)

	flagType = func(key string, values []*registry.Value, path ...string) (string, bool) {
		for _, attr := range values {
			if strings.Join(append(path, attr.Name), "-") == key {
				return attr.Type, true
			}
			if attr.Values != nil {
				typ, found := flagType(key, attr.Values, append(path, attr.Name)...)
				if found {
					return typ, found
				}
			}
		}
		return "", false
	}

	for key, value := range flags {
		ty, found := flagType(key, req.Values)
		if !found {
			return nil, fmt.Errorf("Unknown flag: %v", key)
		}
		parsed, err := coerceValue(ty, value)
		if err != nil {
			return nil, err
		}
		// objx.Set does not create the path,
		// so we do that here
		if strings.Contains(key, "-") {
			parts := strings.Split(key, "-")
			for i, _ := range parts {
				pToCreate := strings.Join(parts[0:i], ".")
				if i > 0 && i < len(parts) && !result.Has(pToCreate) {
					result.Set(pToCreate, map[string]interface{}{})
				}
			}
		}
		path := strings.Replace(key, "-", ".", -1)
		result.Set(path, parsed)
	}

	return result, nil
}

// find a service in a domain matching the name
func serviceWithName(name string) (*registry.Service, error) {
	srvs, err := registry.GetService(name)
	if err == registry.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if len(srvs) == 0 {
		return nil, nil
	}
	return srvs[0], nil
}
