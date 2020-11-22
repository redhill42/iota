package device

import (
	"net/http"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/redhill42/iota/api/types"
	"github.com/redhill42/iota/mqtt"
)

type DeviceManager struct {
	*deviceDB
	publisher *mqtt.Broker
	secret    []byte
	claims    *sync.Map
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

	return &DeviceManager{db, publisher, secret, new(sync.Map)}, nil
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
		return mgr.publisher.Publish(token+"/me/attributes", updates)
	}

	return nil
}

func (mgr *DeviceManager) RPC(id, requestId string, req interface{}) error {
	token, err := mgr.GetToken(id)
	if err != nil {
		return err
	}
	topic := token + "/me/rpc/request/" + requestId
	return mgr.publisher.Publish(topic, req)
}

func (mgr *DeviceManager) Claim(claimId string, attributes map[string]interface{}) error {
	attributes["claim-id"] = claimId
	attributes["claim-time"] = time.Now()

	if _, loaded := mgr.claims.Load(claimId); loaded {
		return DuplicateClaimError(claimId)
	} else {
		mgr.claims.Store(claimId, attributes)
		return nil
	}
}

func (mgr *DeviceManager) GetClaims() []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	mgr.claims.Range(func(key, value interface{}) bool {
		result = append(result, value.(map[string]interface{}))
		return true
	})
	return result
}

func (mgr *DeviceManager) Approve(claimId string, updates map[string]interface{}) (token string, err error) {
	v, loaded := mgr.claims.LoadAndDelete(claimId)
	if !loaded {
		return "", ClaimNotFoundError(claimId)
	}

	// Override claim attributes with approver provided attributes.
	attributes := v.(map[string]interface{})
	for k, v := range updates {
		if v == nil {
			delete(attributes, k)
		} else {
			attributes[k] = v
		}
	}

	// By default, the device id is set to claim id, but the approver can change
	// it by setting the "id" attribute.
	var id = claimId
	delete(attributes, "claim-id")
	delete(attributes, "claim-time")
	if newId, ok := attributes["id"]; ok {
		id = newId.(string)
	}

	if token, err = mgr.CreateToken(id); err != nil {
		return
	}

	// Use Upsert to enable reclaim the device. That is, when a device
	// lost it's access token, it can reclaim. The attributes of reclaimed
	// device is retained.
	err = mgr.Upsert(id, token, attributes)

	// Publish device claim approved message
	topic := "me/claim/" + claimId
	if err == nil {
		err = mgr.publisher.Publish(topic, types.Token{Token: token})
	} else {
		mgr.publisher.Publish(topic, map[string]interface{}{"error": err})
	}
	return
}

func (mgr *DeviceManager) Reject(claimId string) error {
	if _, loaded := mgr.claims.LoadAndDelete(claimId); loaded {
		return mgr.publisher.Publish("me/claim/"+claimId, map[string]string{"error": "Rejected"})
	} else {
		return ClaimNotFoundError(claimId)
	}
}
