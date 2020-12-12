package auth

import (
	"testing"
)

func TestVerify(t *testing.T) {
	srvResource := &Resource{
		Type:     "service",
		Name:     "go.micro.service.foo",
		Endpoint: "Foo.Bar",
	}

	webResource := &Resource{
		Type:     "service",
		Name:     "go.micro.web.foo",
		Endpoint: "/foo/bar",
	}

	catchallResource := &Resource{
		Type:     "*",
		Name:     "*",
		Endpoint: "*",
	}

	tt := []struct {
		Name     string
		Rules    []*Rule
		Account  *Account
		Resource *Resource
		Error    error
	}{
		{
			Name:     "NoRules",
			Rules:    []*Rule{},
			Account:  nil,
			Resource: srvResource,
			Error:    ErrForbidden,
		},
		{
			Name:     "CatchallPublicAccount",
			Account:  &Account{},
			Resource: srvResource,
			Rules: []*Rule{
				&Rule{
					Scope:    "",
					Resource: catchallResource,
				},
			},
		},
		{
			Name:     "CatchallPublicNoAccount",
			Resource: srvResource,
			Rules: []*Rule{
				&Rule{
					Scope:    "",
					Resource: catchallResource,
				},
			},
		},
		{
			Name:     "CatchallPrivateAccount",
			Account:  &Account{},
			Resource: srvResource,
			Rules: []*Rule{
				&Rule{
					Scope:    "*",
					Resource: catchallResource,
				},
			},
		},
		{
			Name:     "CatchallPrivateNoAccount",
			Resource: srvResource,
			Rules: []*Rule{
				&Rule{
					Scope:    "*",
					Resource: catchallResource,
				},
			},
			Error: ErrForbidden,
		},
		{
			Name:     "CatchallServiceRuleMatch",
			Resource: srvResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope: "*",
					Resource: &Resource{
						Type:     srvResource.Type,
						Name:     srvResource.Name,
						Endpoint: "*",
					},
				},
			},
		},
		{
			Name:     "CatchallServiceRuleNoMatch",
			Resource: srvResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope: "*",
					Resource: &Resource{
						Type:     srvResource.Type,
						Name:     "wrongname",
						Endpoint: "*",
					},
				},
			},
			Error: ErrForbidden,
		},
		{
			Name:     "ExactRuleValidScope",
			Resource: srvResource,
			Account: &Account{
				Scopes: []string{"neededscope"},
			},
			Rules: []*Rule{
				&Rule{
					Scope:    "neededscope",
					Resource: srvResource,
				},
			},
		},
		{
			Name:     "ExactRuleInvalidScope",
			Resource: srvResource,
			Account: &Account{
				Scopes: []string{"neededscope"},
			},
			Rules: []*Rule{
				&Rule{
					Scope:    "invalidscope",
					Resource: srvResource,
				},
			},
			Error: ErrForbidden,
		},
		{
			Name:     "CatchallDenyWithAccount",
			Resource: srvResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope:    "*",
					Resource: catchallResource,
					Access:   AccessDenied,
				},
			},
			Error: ErrForbidden,
		},
		{
			Name:     "CatchallDenyWithNoAccount",
			Resource: srvResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope:    "*",
					Resource: catchallResource,
					Access:   AccessDenied,
				},
			},
			Error: ErrForbidden,
		},
		{
			Name:     "RulePriorityGrantFirst",
			Resource: srvResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope:    "*",
					Resource: catchallResource,
					Access:   AccessGranted,
					Priority: 1,
				},
				&Rule{
					Scope:    "*",
					Resource: catchallResource,
					Access:   AccessDenied,
					Priority: 0,
				},
			},
		},
		{
			Name:     "RulePriorityDenyFirst",
			Resource: srvResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope:    "*",
					Resource: catchallResource,
					Access:   AccessGranted,
					Priority: 0,
				},
				&Rule{
					Scope:    "*",
					Resource: catchallResource,
					Access:   AccessDenied,
					Priority: 1,
				},
			},
			Error: ErrForbidden,
		},
		{
			Name:     "WebExactEndpointValid",
			Resource: webResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope:    "*",
					Resource: webResource,
				},
			},
		},
		{
			Name:     "WebExactEndpointInalid",
			Resource: webResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope: "*",
					Resource: &Resource{
						Type:     webResource.Type,
						Name:     webResource.Name,
						Endpoint: "invalidendpoint",
					},
				},
			},
			Error: ErrForbidden,
		},
		{
			Name:     "WebWildcardEndpoint",
			Resource: webResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope: "*",
					Resource: &Resource{
						Type:     webResource.Type,
						Name:     webResource.Name,
						Endpoint: "*",
					},
				},
			},
		},
		{
			Name:     "WebWildcardPathEndpointValid",
			Resource: webResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope: "*",
					Resource: &Resource{
						Type:     webResource.Type,
						Name:     webResource.Name,
						Endpoint: "/foo/*",
					},
				},
			},
		},
		{
			Name:     "WebWildcardPathEndpointInvalid",
			Resource: webResource,
			Account:  &Account{},
			Rules: []*Rule{
				&Rule{
					Scope: "*",
					Resource: &Resource{
						Type:     webResource.Type,
						Name:     webResource.Name,
						Endpoint: "/bar/*",
					},
				},
			},
			Error: ErrForbidden,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			if err := Verify(tc.Rules, tc.Account, tc.Resource); err != tc.Error {
				t.Errorf("Expected %v but got %v", tc.Error, err)
			}
		})
	}
}
