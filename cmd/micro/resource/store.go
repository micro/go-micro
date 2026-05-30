package resource

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/store"
)

// storeCommand exposes the store interface: read, write, delete, list.
func storeCommand() *cli.Command {
	return &cli.Command{
		Name:  "store",
		Usage: "Read and write records in the store",
		Description: `Interact with the data store.

  micro store list [prefix]          List keys (optionally by prefix)
  micro store read <key>             Read a record
  micro store write <key> <value>    Write a record
  micro store delete <key>           Delete a record`,
		Subcommands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List keys",
				ArgsUsage: "[prefix]",
				Action:    storeList,
			},
			{
				Name:      "read",
				Usage:     "Read a record",
				ArgsUsage: "<key>",
				Action:    storeRead,
			},
			{
				Name:      "write",
				Usage:     "Write a record",
				ArgsUsage: "<key> <value>",
				Action:    storeWrite,
			},
			{
				Name:      "delete",
				Usage:     "Delete a record",
				ArgsUsage: "<key>",
				Action:    storeDelete,
			},
		},
	}
}

func storeList(c *cli.Context) error {
	var opts []store.ListOption
	if prefix := c.Args().First(); prefix != "" {
		opts = append(opts, store.ListPrefix(prefix))
	}
	keys, err := store.DefaultStore.List(opts...)
	if err != nil {
		return fail("list: %v", err)
	}
	return printJSON(keys)
}

func storeRead(c *cli.Context) error {
	key := c.Args().First()
	if key == "" {
		return fail("usage: micro store read <key>")
	}
	records, err := store.DefaultStore.Read(key)
	if err != nil {
		return fail("read %q: %v", key, err)
	}
	if len(records) == 0 {
		return fail("key %q not found", key)
	}
	// Print the raw value for a single record, JSON for multiple.
	if len(records) == 1 {
		fmt.Println(string(records[0].Value))
		return nil
	}
	return printJSON(records)
}

func storeWrite(c *cli.Context) error {
	key := c.Args().Get(0)
	value := c.Args().Get(1)
	if key == "" {
		return fail("usage: micro store write <key> <value>")
	}
	rec := &store.Record{Key: key, Value: []byte(value)}
	if err := store.DefaultStore.Write(rec); err != nil {
		return fail("write %q: %v", key, err)
	}
	fmt.Printf("Wrote %q\n", key)
	return nil
}

func storeDelete(c *cli.Context) error {
	key := c.Args().First()
	if key == "" {
		return fail("usage: micro store delete <key>")
	}
	if err := store.DefaultStore.Delete(key); err != nil {
		return fail("delete %q: %v", key, err)
	}
	fmt.Printf("Deleted %q\n", key)
	return nil
}
