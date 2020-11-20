package device

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"net/http"
)

type DeviceManager struct {
	*deviceDB
	secret []byte
}

func NewDeviceManager() (*DeviceManager, error) {
	db, err := openDatabase()
	if err != nil {
		return nil, err
	}

	secret, err := db.getSecret("device")
	if err != nil {
		return nil, err
	}

	return &DeviceManager{db, secret}, nil
}

// CreateToken create an access token for the device. The access token
// can be used by device for further operations.
func (mgr *DeviceManager) CreateToken(id string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		Subject: id,
	})
	return token.SignedString(mgr.secret)
}

func (mgr *DeviceManager) Verify(r *http.Request) (string, error) {
	var claims jwt.StandardClaims

	// Get token from request
	_, err := request.ParseFromRequestWithClaims(r, request.AuthorizationHeaderExtractor, &claims,
		func(token *jwt.Token) (interface{}, error) {
			return mgr.secret, nil
		})
	return claims.Subject, err
}
