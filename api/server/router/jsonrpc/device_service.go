package jsonrpc

import (
	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/device"
)

type DeviceService struct {
	mgr *device.Manager
}

func newDeviceService(ag *agent.Agent) *DeviceService {
	return &DeviceService{ag.DeviceManager}
}

func (s *DeviceService) Create(id string, attributes device.Record) (token string, err error) {
	if token, err = s.mgr.CreateToken(id); err == nil {
		err = s.mgr.Create(id, token, attributes)
	}
	return token, err
}

func (s *DeviceService) Get(id string, keys *[]string) (device.Record, error) {
	if keys == nil {
		return s.mgr.Find(id, nil)
	} else {
		return s.mgr.Find(id, *keys)
	}
}

func (s *DeviceService) Update(id string, updates device.Record) (interface{}, error) {
	return nil, s.mgr.Update(id, updates)
}

func (s *DeviceService) Delete(id string) (interface{}, error) {
	return nil, s.mgr.Remove(id)
}

func (s *DeviceService) List(keys *[]string) ([]device.Record, error) {
	if keys == nil {
		return s.mgr.FindAll(nil)
	} else {
		return s.mgr.FindAll(*keys)
	}
}

func (s *DeviceService) GetClaims() ([]device.Record, error) {
	return s.mgr.GetClaims(), nil
}

func (s *DeviceService) Approve(claimId string, updates device.Record) (string, error) {
	return s.mgr.Approve(claimId, updates)
}

func (s *DeviceService) Reject(claimId string) (interface{}, error) {
	return nil, s.mgr.Reject(claimId)
}
