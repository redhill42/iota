package device

import (
	"encoding/json"
	"net/http"
	"bytes"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/redhill42/iota/mqtt"
)

type DeviceManager struct {
	*deviceDB
	publisher *mqtt.Broker
	secret    []byte
}

func NewDeviceManager(publisher *mqtt.Broker) (*DeviceManager, error) {
	db, err := openDatabase()
	if err != nil {
		return nil, err
	}

	secret, err := db.getSecret("device")
	if err != nil {
		return nil, err
	}

	return &DeviceManager{db, publisher, secret}, nil
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

func (mgr *DeviceManager) Update(id string, updates map[string]interface{}) error {
	err := mgr.deviceDB.Update(id, updates)
	if err != nil {
		return err
	}

	// Publish device attribute updates to device
	if mgr.publisher != nil {
		token, err := mgr.GetToken(id)
		if err != nil {
			return err
		}
		message, err := json.Marshal(updates)
		if err != nil {
			return err
		}
		mgr.publisher.Publish(token+"/me/attributes", message)
	}

	return nil
}

func (mgr *DeviceManager) RPC(id, requestId string, req interface{}) error {
	token, err := mgr.GetToken(id)
	if err != nil {
		return err
	}

	switch req.(type) {
	case string, []byte, bytes.Buffer:
		// message type is ok
	default:
		// must encode to json string
		if req, err = json.Marshal(req); err != nil {
			return err
		}
	}

	topic := token + "/me/rpc/request/" + requestId
	mgr.publisher.Publish(topic, req)
	return nil
}
