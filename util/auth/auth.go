package auth

import (
	"fmt"
	"time"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/logger"
)

// Generate generates a service account for and continually
// refreshes the access token.
func Generate(id string, name string, a auth.Auth) error {
	// extract the account creds from options, these can be set by flags
	accID := a.Options().ID
	accSecret := a.Options().Secret

	// if no credentials were provided, generate an account
	if len(accID) == 0 || len(accSecret) == 0 {
		name := fmt.Sprintf("%v-%v", name, id)

		opts := []auth.GenerateOption{
			auth.WithType("service"),
			auth.WithScopes("service"),
		}

		acc, err := a.Generate(name, opts...)
		if err != nil {
			return err
		}
		logger.Debugf("Auth [%v] Authenticated as %v issued by %v", a, name, acc.Issuer)

		accID = acc.ID
		accSecret = acc.Secret
	}

	// generate the first token
	token, err := a.Token(
		auth.WithCredentials(accID, accSecret),
		auth.WithExpiry(time.Minute*10),
	)
	if err != nil {
		return err
	}

	// set the credentials and token in auth options
	a.Init(
		auth.ClientToken(token),
		auth.Credentials(accID, accSecret),
	)

	// periodically check to see if the token needs refreshing
	go func() {
		timer := time.NewTicker(time.Second * 15)

		for {
			<-timer.C

			// don't refresh the token if it's not close to expiring
			tok := a.Options().Token
			if tok.Expiry.Unix() > time.Now().Add(time.Minute).Unix() {
				continue
			}

			// generate the first token
			tok, err := a.Token(
				auth.WithToken(tok.RefreshToken),
				auth.WithExpiry(time.Minute*10),
			)
			if err != nil {
				logger.Warnf("[Auth] Error refreshing token: %v", err)
				continue
			}

			// set the token
			a.Init(auth.ClientToken(tok))
		}
	}()

	return nil
}
