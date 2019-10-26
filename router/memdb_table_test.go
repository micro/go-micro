package router

func testMemDBTableSetup() (*memDBTable, Route) {
	table := NewMemDBTable()

	route := Route{
		Service: "dest.svc",
		Gateway: "dest.gw",
		Network: "dest.network",
		Router:  "src.router",
		Link:    "det.link",
		Metric:  10,
	}

	return table, route
}
