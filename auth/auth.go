package auth

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/redhill42/iota/auth/userdb"
	"github.com/sirupsen/logrus"
)

const _TOKEN_EXPIRE_TIME = time.Hour * 24 * 30 // 30 days

// The authenticator authenticate user via http protocol
type Authenticator struct {
	db     *userdb.UserDatabase
	secret []byte
}

func NewAuthenticator(db *userdb.UserDatabase) (*Authenticator, error) {
	secret, err := db.GetSecret("jwt")
	if err != nil {
		return nil, err
	}
	return &Authenticator{db, secret}, nil
}

// Authenticate user with name and password. Returns the User object
// and a token
func (auth *Authenticator) Authenticate(username, password string) (*userdb.BasicUser, string, error) {
	// Authenticate user by user database
	user, err := auth.db.Authenticate(username, password)
	if err != nil {
		return nil, "", err
	}

	// Create a new token object, specifying singing method and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		ExpiresAt: time.Now().Add(_TOKEN_EXPIRE_TIME).Unix(),
		Subject:   user.Name,
	})

	// Sign and get the complete encoded token as a string using the secret
	logrus.Debugf("Authenticated user: %v", user.Name)
	tokenString, err := token.SignedString(auth.secret)
	return user, tokenString, err
}

// Verify the current http request is authorized.
func (auth *Authenticator) Verify(r *http.Request) (*userdb.BasicUser, error) {
	var claims jwt.StandardClaims

	// Get token from request
	_, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
		func(token *jwt.Token) (interface{}, error) {
			return auth.secret, nil
		}, request.WithClaims(&claims))

	// If the token is missing or invalid, return error
	if err != nil {
		return nil, err
	}

	return &userdb.BasicUser{Name: claims.Subject}, nil
}

func (auth *Authenticator) VerifyToken(token string) (string, error) {
	var claims jwt.StandardClaims
	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return auth.secret, nil
	})
	return claims.Subject, err
}
