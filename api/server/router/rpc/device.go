package rpc

import (
	"net/http"

	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/device"
)

type DeviceService struct {
	mgr *device.Manager
}

func newDeviceService(ag *agent.Agent) *DeviceService {
	return &DeviceService{ag.DeviceManager}
}

type ListArgs struct {
	Keys []string `json:"keys"`
}

func (s *DeviceService) List(r *http.Request, args *ListArgs, result *[]device.Record) (err error) {
	*result, err = s.mgr.FindAll(args.Keys)
	return err
}

type CreateArgs struct {
	Id         string        `json:"id"`
	Attributes device.Record `json:"attributes"`
}

func (s *DeviceService) Create(r *http.Request, args *CreateArgs, result *string) error {
	token, err := s.mgr.CreateToken(args.Id)
	if err != nil {
		return err
	}
	if err = s.mgr.Create(args.Id, token, args.Attributes); err == nil {
		*result = token
	}
	return err
}

type GetArgs struct {
	Id   string   `json:"id"`
	Keys []string `json:"keys"`
}

func (s *DeviceService) Get(r *http.Request, args *GetArgs, result *device.Record) (err error) {
	*result, err = s.mgr.Find(args.Id, args.Keys)
	return err
}

type UpdateArgs struct {
	Id      string        `json:"id"`
	Updates device.Record `json:"updates"`
}

func (s *DeviceService) Update(r *http.Request, args *UpdateArgs, result *interface{}) error {
	return s.mgr.Update(args.Id, args.Updates)
}

type DeleteArgs struct {
	Id string `json:"id"`
}

func (s *DeviceService) Delete(r *http.Request, args *DeleteArgs, result *interface{}) error {
	return s.mgr.Remove(args.Id)
}

type GetClaimsArgs struct {
}

func (s *DeviceService) GetClaims(r *http.Request, args *GetClaimsArgs, result *[]device.Record) error {
	*result = s.mgr.GetClaims()
	return nil
}

type ApproveArgs struct {
	ClaimId string        `json:"claimId"`
	Updates device.Record `json:"updates"`
}

func (s *DeviceService) Approve(r *http.Request, args *ApproveArgs, result *string) (err error) {
	*result, err = s.mgr.Approve(args.ClaimId, args.Updates)
	return err
}

type RejectArgs struct {
	ClaimId string `json:"claimId"`
}

func (s *DeviceService) Reject(r *http.Request, args *RejectArgs, result *interface{}) error {
	return s.mgr.Reject(args.ClaimId)
}
