package router

import "testing"

func TestHash(t *testing.T) {
	route1 := Route{
		Service: "dest.svc",
		Gateway: "dest.gw",
		Network: "dest.network",
		Link:    "det.link",
		Metric:  10,
	}

	// make a copy
	route2 := route1

	route1Hash := route1.Hash()
	route2Hash := route2.Hash()

	// we should get the same hash
	if route1Hash != route2Hash {
		t.Errorf("identical routes result in different hashes")
	}
}
