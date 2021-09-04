package main

import (
	"fmt"
	"os"

	"github.com/asim/go-micro/v3"
	"github.com/urfave/cli/v2"
)

func main() {
	service := micro.NewService(
		// Add runtime flags
		// We could do this below too
		micro.Flags(
			&cli.StringFlag{
				Name:  "string_flag",
				Usage: "This is a string flag",
			},
			&cli.IntFlag{
				Name:  "int_flag",
				Usage: "This is an int flag",
			},
			&cli.BoolFlag{
				Name:  "bool_flag",
				Usage: "This is a bool flag",
			},
		),
	)

	// Init will parse the command line flags. Any flags set will
	// override the above settings. Options defined here will
	// override anything set on the command line.
	service.Init(
		// Add runtime action
		// We could actually do this above
		micro.Action(func(c *cli.Context) error {
			fmt.Printf("The string flag is: %s\n", c.String("string_flag"))
			fmt.Printf("The int flag is: %d\n", c.Int("int_flag"))
			fmt.Printf("The bool flag is: %t\n", c.Bool("bool_flag"))
			// let's just exit because
			os.Exit(0)
			return nil
		}),
	)

	// Run the server
	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
