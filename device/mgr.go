package device

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/api/types"
	"github.com/redhill42/iota/config"
	"github.com/redhill42/iota/mqtt"
)

type Manager struct {
	*deviceDB
	broker       *mqtt.Broker
	secret       []byte
	claims       *sync.Map
	autoapprove  bool
	rpcTimeout   time.Duration
	rpcRequestId int64
}

func NewManager(broker *mqtt.Broker) (*Manager, error) {
	db, err := openDatabase()
	if err != nil {
		return nil, err
	}

	secret, err := db.getSecret("device")
	if err != nil {
		return nil, err
	}

	autoapprove, _ := strconv.ParseBool(config.GetOrDefault("device.autoapprove", "false"))
	rpcTimeout, _ := strconv.ParseInt(config.GetOrDefault("device.rpcTimeout", "5"), 10, 0)

	return &Manager{
		deviceDB:     db,
		broker:       broker,
		secret:       secret,
		claims:       new(sync.Map),
		autoapprove:  autoapprove,
		rpcTimeout:   time.Duration(rpcTimeout) * time.Second,
		rpcRequestId: time.Now().Unix(),
	}, nil
}

// CreateToken create an access token for the device. The access token
// can be used by device for further operations.
func (mgr *Manager) CreateToken(id string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		Subject: id,
	})
	return token.SignedString(mgr.secret)
}

func (mgr *Manager) Verify(r *http.Request) (string, error) {
	var claims jwt.StandardClaims

	// Get token from request
	_, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
		func(token *jwt.Token) (interface{}, error) {
			return mgr.secret, nil
		}, request.WithClaims(&claims))
	return claims.Subject, err
}

func (mgr *Manager) VerifyToken(token string) (string, error) {
	var claims jwt.StandardClaims
	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return mgr.secret, nil
	})
	return claims.Subject, err
}

func (mgr *Manager) Update(id string, updates Record) error {
	err := mgr.deviceDB.Update(id, updates)
	if err != nil {
		return err
	}

	// Publish device attribute updates to device
	if len(updates) != 0 && mgr.broker != nil {
		token, err := mgr.GetToken(id)
		if err != nil {
			return err
		}
		updates["id"] = id
		return mgr.broker.Publish(token+"/me/attributes", updates)
	}

	return nil
}

type RPCRequest struct {
	Version string      `json:"jsonrpc,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	Id      interface{} `json:"id,omitempty"`
}

// isBatch returns true when the first non-whitespace character is '['
func isBatch(raw []byte) bool {
	for _, c := range raw {
		// skip insignificant whitespace
		if c == 0x20 || c == 0x09 || c == 0xa || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

// parseRPCRequest parse raw message. Returns true if the request
// needs response, false otherwise.
func parseRPCRequest(raw []byte) (bool, error) {
	if isBatch(raw) {
		dec := json.NewDecoder(bytes.NewReader(raw))
		if _, err := dec.Token(); err != nil { // skip  '['
			return false, err
		}
		for dec.More() {
			var msg RPCRequest
			if err := dec.Decode(&msg); err != nil {
				return false, err
			}
			if msg.Id != nil {
				return true, nil
			}
		}
		return false, nil
	} else {
		var msg RPCRequest
		err := json.Unmarshal(raw, &msg)
		return msg.Id != nil, err
	}
}

func (mgr *Manager) RPC(id string, req []byte) ([]byte, error) {
	needResponse, err := parseRPCRequest(req)
	if err != nil {
		return nil, httputils.NewStatusError(http.StatusBadRequest, err)
	}

	token, err := mgr.GetToken(id)
	if err != nil {
		return nil, err
	}

	requestId := strconv.FormatInt(atomic.AddInt64(&mgr.rpcRequestId, 1), 10)
	requestTopic := token + "/me/rpc/request/" + requestId
	responseTopic := token + "/me/rpc/response/" + requestId

	if !needResponse || mgr.rpcTimeout <= 0 {
		return nil, mgr.broker.Publish(requestTopic, req)
	}

	// Subscribe on response
	respCh := make(chan []byte)
	err = mgr.broker.Subscribe(responseTopic, func(topic string, message []byte) {
		respCh <- message
	})
	if err != nil {
		return nil, err
	}
	defer mgr.broker.Unsubscribe(responseTopic)

	// Send request to device
	if err = mgr.broker.Publish(requestTopic, req); err != nil {
		return nil, err
	}

	// Receive response or time out
	select {
	case <-time.After(mgr.rpcTimeout):
		return nil, RPCTimeoutError(id)
	case msg := <-respCh:
		return msg, nil
	}
}

func (mgr *Manager) Claim(claimId string, attributes Record) error {
	if !validateDeviceId(claimId) {
		return InvalidDeviceIdError(claimId)
	}
	if attributes == nil {
		attributes = make(Record)
	}
	attributes["claim-id"] = claimId
	attributes["claim-time"] = time.Now()

	if _, loaded := mgr.claims.Load(claimId); loaded {
		return DuplicateClaimError(claimId)
	}
	if mgr.autoapprove {
		_, err := mgr.internalApprove(claimId, attributes)
		return err
	} else {
		mgr.claims.Store(claimId, attributes)
		return nil
	}
}

func (mgr *Manager) GetClaims() []Record {
	result := make([]Record, 0)
	mgr.claims.Range(func(key, value interface{}) bool {
		result = append(result, value.(Record))
		return true
	})
	return result
}

func (mgr *Manager) Approve(claimId string, updates Record) (token string, err error) {
	v, loaded := mgr.claims.LoadAndDelete(claimId)
	if !loaded {
		return "", ClaimNotFoundError(claimId)
	}

	// Override claim attributes with approver provided attributes.
	attributes := v.(Record)
	for k, v := range updates {
		if v == nil {
			delete(attributes, k)
		} else {
			attributes[k] = v
		}
	}

	return mgr.internalApprove(claimId, attributes)
}

func (mgr *Manager) internalApprove(claimId string, attributes Record) (token string, err error) {
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
		err = mgr.broker.Publish(topic, types.Token{Token: token})
	} else {
		_ = mgr.broker.Publish(topic, map[string]interface{}{"error": err})
	}
	return
}

func (mgr *Manager) Reject(claimId string) error {
	if _, loaded := mgr.claims.LoadAndDelete(claimId); loaded {
		return mgr.broker.Publish("me/claim/"+claimId, map[string]string{"error": "Rejected"})
	} else {
		return ClaimNotFoundError(claimId)
	}
}
