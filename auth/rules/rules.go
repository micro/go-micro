package rules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/micro/go-micro/v2/auth"
)

// Verify an account has access to a resource using the rules provided. If the account does not have
// access an error will be returned. If there are no rules provided which match the resource, an error
// will be returned
func Verify(namespace string, rules []*auth.Rule, acc *auth.Account, res *auth.Resource) error {
	// the rule is only to be applied if the type matches the resource or is catch-all (*)
	validTypes := []string{"*", res.Type}

	// the rule is only to be applied if the name matches the resource or is catch-all (*)
	validNames := []string{"*", res.Name}

	// rules can have wildcard excludes on endpoints since this can also be a path for web services,
	// e.g. /foo/* would include /foo/bar. We also want to check for wildcards and the exact endpoint
	validEndpoints := []string{"*", res.Endpoint}
	if comps := strings.Split(res.Endpoint, "/"); len(comps) > 1 {
		for i := 1; i < len(comps); i++ {
			wildcard := fmt.Sprintf("%v/*", strings.Join(comps[0:i], "/"))
			validEndpoints = append(validEndpoints, wildcard)
		}
	}

	// filter the rules to the ones which match the criteria above
	filteredRules := make([]*auth.Rule, 0)
	for _, rule := range rules {
		if !include(validTypes, rule.Resource.Type) {
			continue
		}
		if !include(validNames, rule.Resource.Name) {
			continue
		}
		if !include(validEndpoints, rule.Resource.Endpoint) {
			continue
		}
		filteredRules = append(filteredRules, rule)
	}

	// sort the filtered rules by priority, highest to lowest
	sort.SliceStable(filteredRules, func(i, j int) bool {
		return filteredRules[i].Priority > filteredRules[j].Priority
	})

	// loop through the rules and check for a rule which applies to this account
	for _, rule := range filteredRules {
		// a blank role indicates the rule applies to everyone, even nil accounts
		if rule.Role == "" && rule.Access == auth.AccessDenied {
			return auth.ErrForbidden
		} else if rule.Role == "" && rule.Access == auth.AccessGranted {
			return nil
		}

		// all further checks require an account within the current scope
		if acc == nil || !acc.HasScope("namespace", namespace) {
			continue
		}

		// this rule applies to any account
		if rule.Role == "*" && rule.Access == auth.AccessDenied {
			return auth.ErrForbidden
		} else if rule.Role == "" && rule.Access == auth.AccessGranted {
			return nil
		}

		// if the account has the necessary role
		if include(acc.Roles, rule.Role) && rule.Access == auth.AccessDenied {
			return auth.ErrForbidden
		} else if rule.Role == "" && rule.Access == auth.AccessGranted {
			return nil
		}
	}

	// if no rules matched then return forbidden
	return auth.ErrForbidden
}

// include is a helper function which checks to see if the slice contains the value. includes is
// not case sensitive.
func include(slice []string, val string) bool {
	for _, s := range slice {
		if strings.ToLower(s) == strings.ToLower(val) {
			return true
		}
	}
	return false
}
