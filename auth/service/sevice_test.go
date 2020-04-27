package service

import (
	"testing"

	pb "github.com/micro/go-micro/v2/auth/service/proto"
)

func TestListRulesSorting(t *testing.T) {
	s := &svc{
		rules: []*pb.Rule{
			&pb.Rule{Priority: 1},
			&pb.Rule{Priority: 3},
			&pb.Rule{Priority: 2},
		},
	}

	var priorities []int32
	for _, r := range s.listRules() {
		priorities = append(priorities, r.Priority)
	}

	if priorities[0] != 1 || priorities[1] != 2 || priorities[2] != 3 {
		t.Errorf("Incorrect Rule Sequence")
	}
}
