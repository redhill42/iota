package alarms

import (
	"net/http"

	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/alarm"
	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/api/server/router"
	"github.com/redhill42/iota/api/server/websocket"
)

const alarmPath = "/alarms/{id:[0-9a-f]+}"

type alarmsRouter struct {
	*agent.Agent
	routes []router.Route
	hub    *websocket.Hub
}

func NewRouter(agent *agent.Agent) router.Router {
	h := websocket.NewHub()
	go h.Run()
	agent.AlarmManager.OnUpdate(func(rec *alarm.Alarm) {
		h.Updates() <- rec
	})

	r := &alarmsRouter{Agent: agent, hub: h}
	r.routes = []router.Route{
		router.NewGetRoute("/alarms", r.list),
		router.NewPostRoute("/alarms", r.upsert),
		router.NewGetRoute(alarmPath, r.read),
		router.NewDeleteRoute(alarmPath, r.delete),
		router.NewPostRoute(alarmPath+"/clear", r.clear),

		router.NewPostRoute("/me/alarm", r.upsertMe),
		router.NewGetRoute("/me/alarm/{name:[^/]+}", r.readMe),
		router.NewDeleteRoute("/me/alarm/{name:[^/]+}", r.deleteMe),
		router.NewPostRoute("/me/alarm/{name:[^/]+}/clear", r.clearMe),

		router.NewGetRoute("/alarms/{id:[0-9a-f]+|\\+}/subscribe", r.subscribe),
	}
	return r
}

func (ar *alarmsRouter) Routes() []router.Route {
	return ar.routes
}

func (ar *alarmsRouter) list(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	result, err := ar.AlarmManager.FindAll()
	if err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, result)
}

func (ar *alarmsRouter) upsert(w http.ResponseWriter, r *http.Request, vars map[string]string) (err error) {
	var rec alarm.Alarm
	if err = httputils.ReadJSON(r, &rec); err != nil {
		return err
	}
	if err = ar.AlarmManager.Upsert(&rec); err != nil {
		return err
	}
	w.Header().Set("Location", r.RequestURI+"/"+rec.ID)
	return httputils.WriteJSON(w, http.StatusOK, map[string]string{"id": rec.ID})
}

func (ar *alarmsRouter) upsertMe(w http.ResponseWriter, r *http.Request, vars map[string]string) (err error) {
	var rec alarm.Alarm
	if err = httputils.ReadJSON(r, &rec); err != nil {
		return err
	}
	rec.Originator = vars["id"]
	if err = ar.AlarmManager.Upsert(&rec); err != nil {
		return err
	}
	w.Header().Set("Location", r.RequestURI+"/"+rec.Name)
	return httputils.WriteJSON(w, http.StatusOK, map[string]string{"id": rec.ID})
}

func (ar *alarmsRouter) read(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	rec, err := ar.AlarmManager.Find(vars["id"])
	if err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, &rec)
}

func (ar *alarmsRouter) readMe(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	rec, err := ar.AlarmManager.FindName(vars["name"], vars["id"])
	if err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, &rec)
}

func (ar *alarmsRouter) delete(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := ar.AlarmManager.Delete(vars["id"]); err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func (ar *alarmsRouter) deleteMe(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := ar.AlarmManager.DeleteName(vars["name"], vars["id"]); err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func (ar *alarmsRouter) clear(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := ar.AlarmManager.Clear(vars["id"]); err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func (ar *alarmsRouter) clearMe(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := ar.AlarmManager.ClearName(vars["name"], vars["id"]); err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func (ar *alarmsRouter) subscribe(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	return ar.hub.ServeWS(w, r, vars["id"])
}
