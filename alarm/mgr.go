package alarm

type UpdateCallback func(alarm *Alarm)

type Manager struct {
	*alarmDB
	updateCallbacks []UpdateCallback
}

func NewManager() (*Manager, error) {
	db, err := openDatabase()
	if err != nil {
		return nil, err
	}
	return &Manager{alarmDB: db}, nil
}

func (mgr *Manager) Upsert(alarm *Alarm) error {
	err := mgr.alarmDB.Upsert(alarm)
	if err != nil {
		return err
	}
	for _, cb := range mgr.updateCallbacks {
		cb(alarm)
	}
	return nil
}

func (mgr *Manager) OnUpdate(callback UpdateCallback) {
	mgr.updateCallbacks = append(mgr.updateCallbacks, callback)
}
