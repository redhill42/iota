package jsonrpc

import (
	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/alarm"
)

type AlarmService struct {
	mgr *alarm.Manager
}

func newAlarmService(ag *agent.Agent) *AlarmService {
	return &AlarmService{ag.AlarmManager}
}

func (s *AlarmService) Upsert(alarm *alarm.Alarm) (string, error) {
	err := s.mgr.Upsert(alarm)
	return alarm.ID, err
}

func (s *AlarmService) Find(id string) (*alarm.Alarm, error) {
	return s.mgr.Find(id)
}

func (s *AlarmService) FindName(name, originator string) (*alarm.Alarm, error) {
	return s.mgr.FindName(name, originator)
}

func (s *AlarmService) FindAll() ([]*alarm.Alarm, error) {
	return s.mgr.FindAll()
}

func (s *AlarmService) Delete(id string) error {
	return s.mgr.Delete(id)
}

func (s *AlarmService) DeleteName(name, originator string) error {
	return s.mgr.DeleteName(name, originator)
}

func (s *AlarmService) Clear(id string) error {
	return s.mgr.Clear(id)
}

func (s *AlarmService) ClearName(name, originator string) error {
	return s.mgr.ClearName(name, originator)
}
