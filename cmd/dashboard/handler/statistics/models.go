package statistics

type getSummaryResponse struct {
	Registry registrySummary `json:"registry"`
	Services servicesSummary `json:"services"`
}

type registrySummary struct {
	Type  string   `json:"type"`
	Addrs []string `json:"addrs"`
}

type servicesSummary struct {
	Count      int `json:"count"`
	NodesCount int `json:"nodes_count"`
}
