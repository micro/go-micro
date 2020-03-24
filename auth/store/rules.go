package store

import (
	"encoding/json"
	"strings"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/store"
)

// Rule is an access control rule
type Rule struct {
	Role     string         `json:"rule"`
	Resource *auth.Resource `json:"resource"`
}

var joinKey = ":"

// Key to be used when written to the store
func (r *Rule) Key() string {
	comps := []string{r.Resource.Type, r.Resource.Name, r.Resource.Endpoint, r.Role}
	return strings.Join(comps, joinKey)
}

// Bytes returns json encoded bytes
func (r *Rule) Bytes() []byte {
	bytes, _ := json.Marshal(r)
	return bytes
}

// isValidRule returns a bool, indicating if a rule permits access to a
// resource for a given account
func isValidRule(rule Rule, acc *auth.Account, res *auth.Resource) bool {
	if rule.Role == "*" {
		return true
	}

	for _, role := range acc.Roles {
		if rule.Role == role {
			return true
		}

		// allow user.anything if role is user.*
		if strings.HasSuffix(rule.Role, ".*") && strings.HasPrefix(rule.Role, role+".") {
			return true
		}
	}

	return false
}

// listRules gets all the rules from the store which have a key
// prefix matching the filters
func (s *Store) listRules(filters ...string) ([]Rule, error) {
	// get the records from the store
	prefix := strings.Join(filters, joinKey)
	recs, err := s.opts.Store.Read(prefix, store.ReadPrefix())
	if err != nil {
		return nil, err
	}

	// unmarshal the records
	rules := make([]Rule, 0, len(recs))
	for _, rec := range recs {
		var r Rule
		if err := json.Unmarshal(rec.Value, &r); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	// return the rules
	return rules, nil
}
