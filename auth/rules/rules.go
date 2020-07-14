package rules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/micro/go-micro/v2/logger"

	"github.com/micro/go-micro/v2/auth"
)

// Verify an account has access to a resource using the rules provided. If the account does not have
// access an error will be returned. If there are no rules provided which match the resource, an error
// will be returned
func Verify(rules []*auth.Rule, acc *auth.Account, res *auth.Resource) error {
	logger.Debugf("Verify using %v rules: %v => %v", len(rules), acc, res)

	// the rule is only to be applied if the type matches the resource or is catch-all (*)
	validTypes := []string{"*", res.Type}

	// the rule is only to be applied if the name matches the resource or is catch-all (*)
	validNames := []string{"*", res.Name}

	// rules can have wildcard excludes on endpoints since this can also be a path for web services,
	// e.g. /foo/* would include /foo/bar. We also want to check for wildcards and the exact endpoint
	validEndpoints := []string{"*", res.Endpoint}
	if comps := strings.Split(res.Endpoint, "/"); len(comps) > 1 {
		for i := 1; i < len(comps)+1; i++ {
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
		// a blank scope indicates the rule applies to everyone, even nil accounts
		if rule.Scope == auth.ScopePublic && rule.Access == auth.AccessDenied {
			logger.Debugf("Access was explicitly denied by a public scope rule")
			return auth.ErrForbidden
		} else if rule.Scope == auth.ScopePublic && rule.Access == auth.AccessGranted {
			logger.Debugf("Access was explicitly granted by a public scope rule")
			return nil
		}

		// all further checks require an account
		if acc == nil {
			logger.Debugf("Rule was skipped as account is nil")
			continue
		}

		// this rule applies to any account
		if rule.Scope == auth.ScopeAccount && rule.Access == auth.AccessDenied {
			logger.Debugf("Access was explicitly denied by a account scope rule")
			return auth.ErrForbidden
		} else if rule.Scope == auth.ScopeAccount && rule.Access == auth.AccessGranted {
			logger.Debugf("Access was explicitly granted by a account scope rule")
			return nil
		}

		// if the account has the necessary scope
		if include(acc.Scopes, rule.Scope) && rule.Access == auth.AccessDenied {
			logger.Debugf("Account was denied due to a lack of role: %v", rule.Scope)
			return auth.ErrForbidden
		} else if include(acc.Scopes, rule.Scope) && rule.Access == auth.AccessGranted {
			logger.Debugf("Account was granted due to a presence of role: %v", rule.Scope)
			return nil
		}
	}

	logger.Debugf("Access was denied because no rules granted access")

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
