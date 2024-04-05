package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/helixml/helix/api/pkg/auth"
	"github.com/helixml/helix/api/pkg/store"
	"github.com/helixml/helix/api/pkg/types"
)

type authMiddleware struct {
	authenticator auth.Authenticator
	options       ServerOptions
	store         store.Store
}

func newMiddleware(authenticator auth.Authenticator, options ServerOptions, store store.Store) *authMiddleware {
	return &authMiddleware{authenticator: authenticator, options: options, store: store}
}

func (auth *authMiddleware) maybeOwnerFromRequest(r *http.Request) (*types.ApiKey, error) {
	// in case the request is authenticated with an hl- token, rather than a
	// keycloak JWT, return the owner. Returns nil if it's not an hl- token.
	token := r.Header.Get("Authorization")
	token = extractBearerToken(token)

	if token == "" {
		token = r.URL.Query().Get("access_token")
	}

	if token == "" {
		return nil, nil
	}

	if strings.HasPrefix(token, types.API_KEY_PREIX) {
		if owner, err := auth.store.CheckAPIKey(r.Context(), token); err != nil {
			return nil, fmt.Errorf("error checking API key: %s", err.Error())
		} else if owner == nil {
			// user claimed to provide hl- token, but it was invalid
			return nil, fmt.Errorf("invalid API key")
		} else {
			return owner, nil
		}
	}
	// user didn't claim token was an lp token, so fallback to keycloak
	return nil, nil
}

func (auth *authMiddleware) jwtFromRequest(r *http.Request) (*jwt.Token, error) {
	// try to extract Authorization parameter from the HTTP header
	token := r.Header.Get("Authorization")
	if token != "" {
		// extract Bearer token
		token = extractBearerToken(token)
		if token == "" {
			return nil, fmt.Errorf("bearer token missing")
		}
	} else {
		// try to extract access_token query parameter
		token = r.URL.Query().Get("access_token")
		if token == "" {
			return nil, fmt.Errorf("token missing")
		}
	}

	return auth.authenticator.ValidateUserToken(r.Context(), token)
}

func (auth *authMiddleware) userIDFromRequest(r *http.Request) (string, error) {
	token, err := auth.jwtFromRequest(r)
	if err != nil {
		return "", err
	}
	return getUserIdFromJWT(token), nil
}

// this will return a user id based on EITHER a database token OR a keycloak token
func (auth *authMiddleware) userIDFromRequestBothModes(r *http.Request) (string, error) {
	databaseToken, err := auth.maybeOwnerFromRequest(r)
	if err != nil {
		return "", err
	}
	if databaseToken != nil {
		return databaseToken.Owner, nil
	}

	authToken, err := auth.jwtFromRequest(r)
	if err != nil {
		return "", err
	}
	return getUserIdFromJWT(authToken), nil
}

func getUserFromJWT(tok *jwt.Token) types.UserData {
	if tok == nil {
		return types.UserData{}
	}
	mc := tok.Claims.(jwt.MapClaims)
	uid := mc["sub"].(string)
	email := mc["email"].(string)
	name := mc["name"].(string)
	return types.UserData{
		ID:       uid,
		Email:    email,
		FullName: name,
	}
}

func getUserIdFromJWT(tok *jwt.Token) string {
	user := getUserFromJWT(tok)
	return user.ID
}

func setRequestUser(ctx context.Context, user types.UserData) context.Context {
	ctx = context.WithValue(ctx, "userid", user.ID)
	ctx = context.WithValue(ctx, "email", user.Email)
	ctx = context.WithValue(ctx, "fullname", user.FullName)
	return ctx
}

func getRequestUser(req *http.Request) types.UserData {
	id := req.Context().Value("userid")
	email := req.Context().Value("email")
	fullname := req.Context().Value("fullname")
	return types.UserData{
		ID:       id.(string),
		Email:    email.(string),
		FullName: fullname.(string),
	}
}

// this happens in the very first middleware to populate the request context
// based on EITHER the database api token OR then the keycloak JWT
func (auth *authMiddleware) verifyToken(next http.Handler, enforce bool) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		maybeOwner, err := auth.maybeOwnerFromRequest(r)
		if err != nil && enforce {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if maybeOwner == nil {
			// check keycloak JWT
			token, err := auth.jwtFromRequest(r)
			if err != nil && enforce {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			r = r.WithContext(setRequestUser(r.Context(), getUserFromJWT(token)))
			next.ServeHTTP(w, r)
			return
		}
		// successful api_key auth
		r = r.WithContext(setRequestUser(r.Context(), types.UserData{
			ID: maybeOwner.Owner,
		}))
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}

func (auth *authMiddleware) maybeVerifyToken(next http.Handler) http.Handler {
	return auth.verifyToken(next, false)
}

func (auth *authMiddleware) enforceVerifyToken(next http.Handler) http.Handler {
	return auth.verifyToken(next, true)
}

func (auth *authMiddleware) apiKeyAuth(f http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		maybeOwner, err := auth.maybeOwnerFromRequest(req)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusUnauthorized)
			return
		}
		// successful api_key auth
		req = req.WithContext(setRequestUser(req.Context(), types.UserData{
			ID: maybeOwner.Owner,
		}))
		f.ServeHTTP(rw, req)
	}
}
