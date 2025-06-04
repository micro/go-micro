package natsjskv

import "go-micro.dev/v5/store"

type test struct {
	Record   *store.Record
	Database string
	Table    string
}

var (
	table = []test{
		{
			Record: &store.Record{
				Key:   "One",
				Value: []byte("First value"),
			},
		},
		{
			Record: &store.Record{
				Key:   "Two",
				Value: []byte("Second value"),
			},
			Table: "prefix_test",
		},
		{
			Record: &store.Record{
				Key:   "Third",
				Value: []byte("Third value"),
			},
			Database: "new-bucket",
		},
		{
			Record: &store.Record{
				Key:   "Four",
				Value: []byte("Fourth value"),
			},
			Database: "new-bucket",
			Table:    "prefix_test",
		},
		{
			Record: &store.Record{
				Key:   "empty-value",
				Value: []byte{},
			},
			Database: "new-bucket",
		},
		{
			Record: &store.Record{
				Key:   "Alex",
				Value: []byte("Some value"),
			},
			Database: "prefix-test",
			Table:    "names",
		},
		{
			Record: &store.Record{
				Key:   "Jones",
				Value: []byte("Some value"),
			},
			Database: "prefix-test",
			Table:    "names",
		},
		{
			Record: &store.Record{
				Key:   "Adrianna",
				Value: []byte("Some value"),
			},
			Database: "prefix-test",
			Table:    "names",
		},
		{
			Record: &store.Record{
				Key:   "MexicoCity",
				Value: []byte("Some value"),
			},
			Database: "prefix-test",
			Table:    "cities",
		},
		{
			Record: &store.Record{
				Key:   "HoustonCity",
				Value: []byte("Some value"),
			},
			Database: "prefix-test",
			Table:    "cities",
		},
		{
			Record: &store.Record{
				Key:   "ZurichCity",
				Value: []byte("Some value"),
			},
			Database: "prefix-test",
			Table:    "cities",
		},
		{
			Record: &store.Record{
				Key:   "Helsinki",
				Value: []byte("Some value"),
			},
			Database: "prefix-test",
			Table:    "cities",
		},
		{
			Record: &store.Record{
				Key:   "testKeytest",
				Value: []byte("Some value"),
			},
			Table: "some_table",
		},
		{
			Record: &store.Record{
				Key:   "testSecondtest",
				Value: []byte("Some value"),
			},
			Table: "some_table",
		},
		{
			Record: &store.Record{
				Key:   "lalala",
				Value: []byte("Some value"),
			},
			Table: "some_table",
		},
		{
			Record: &store.Record{
				Key:   "testAnothertest",
				Value: []byte("Some value"),
			},
		},
		{
			Record: &store.Record{
				Key:   "FobiddenCharactersAreAllowed:|@..+",
				Value: []byte("data no matter"),
			},
		},
	}
)
