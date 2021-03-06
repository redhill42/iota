package devices

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/api/server/router"
	"github.com/redhill42/iota/api/server/websocket"
	"github.com/redhill42/iota/api/types"
	"github.com/redhill42/iota/device"
)

const devicePath = "/devices/{id:[^/]+}"
const claimPath = "/claims/{id:[^/]+}"

type devicesRouter struct {
	*agent.Agent
	routes []router.Route
	hub    *websocket.Hub
}

func NewRouter(agent *agent.Agent) router.Router {
	h := websocket.NewHub()
	go h.Run()
	agent.DeviceManager.OnUpdate(func(rec device.Record) {
		h.Updates() <- rec
	})

	r := &devicesRouter{Agent: agent, hub: h}
	r.routes = []router.Route{
		router.NewGetRoute("/devices", r.list),
		router.NewPostRoute("/devices", r.create),
		router.NewGetRoute(devicePath, r.read),
		router.NewPutRoute(devicePath, r.update),
		router.NewDeleteRoute(devicePath, r.delete),
		router.NewPostRoute(devicePath+"/rpc", r.rpc),

		router.NewGetRoute(devicePath+"/subscribe", r.subscribe),

		router.NewGetRoute("/claims", r.getClaims),
		router.NewPostRoute(claimPath+"/approve", r.approve),
		router.NewPostRoute(claimPath+"/reject", r.reject),

		router.NewPostRoute("/me/claim", r.claim),
		router.NewGetRoute("/me/attributes", r.read),
		router.NewPostRoute("/me/attributes", r.update),
		router.NewPostRoute("/me/measurement", r.measurement),
	}
	return r
}

func (dr *devicesRouter) Routes() []router.Route {
	return dr.routes
}

func (dr *devicesRouter) list(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var keys []string
	if r.FormValue("keys") != "" {
		keys = strings.Split(r.FormValue("keys"), ",")
	}
	result, err := dr.DeviceManager.FindAll(keys)
	if err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, result)
}

func (dr *devicesRouter) create(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var (
		req       device.Record
		id, token string
		ok        bool
		err       error
	)

	if err = httputils.ReadJSON(r, &req); err != nil {
		return err
	}
	if id, ok = req["id"].(string); !ok {
		http.Error(w, "Missing \"id\" attribute", http.StatusBadRequest)
		return nil
	}
	if token, err = dr.DeviceManager.CreateToken(id); err != nil {
		return err
	}
	if err = dr.DeviceManager.Create(id, token, req); err != nil {
		return err
	}
	w.Header().Set("Location", r.RequestURI+"/"+id)
	return httputils.WriteJSON(w, http.StatusCreated, types.Token{Token: token})
}

func (dr *devicesRouter) read(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var keys []string
	if r.FormValue("keys") != "" {
		keys = strings.Split(r.FormValue("keys"), ",")
	}
	info, err := dr.DeviceManager.Find(vars["id"], keys)
	if err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, &info)
}

func (dr *devicesRouter) update(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var (
		req device.Record
		err error
	)
	if err = httputils.ReadJSON(r, &req); err != nil {
		return err
	}
	if err = dr.DeviceManager.Update(vars["id"], req); err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func (dr *devicesRouter) delete(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := dr.DeviceManager.Remove(vars["id"]); err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func (dr *devicesRouter) rpc(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	req, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	resp, err := dr.DeviceManager.RPC(r.Context(), vars["id"], req)
	switch {
	case err != nil:
		return err
	case resp == nil:
		w.WriteHeader(http.StatusNoContent)
	default:
		_, err = w.Write(resp)
	}
	return err
}

func (dr *devicesRouter) subscribe(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	return dr.hub.ServeWS(w, r, vars["id"])
}

func (dr *devicesRouter) measurement(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	// Add device id tag
	record := strings.Split(string(body), " ")
	record[0] += ",device=" + vars["id"]

	// Write record to time series database
	dr.TSDB.WriteRecord(strings.Join(record, " "))
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (dr *devicesRouter) claim(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var (
		req device.Record
		id  string
		ok  bool
		err error
	)

	if err = httputils.ReadJSON(r, &req); err != nil {
		return err
	}
	if id, ok = req["claim-id"].(string); !ok {
		http.Error(w, "Missing \"claim-id\" attributes", http.StatusBadRequest)
		return nil
	}
	if err = dr.DeviceManager.Claim(id, req); err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusAccepted)
		return nil
	}
}

func (dr *devicesRouter) getClaims(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	claims := dr.DeviceManager.GetClaims()
	return httputils.WriteJSON(w, http.StatusOK, claims)
}

func (dr *devicesRouter) approve(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var updates device.Record
	var id = vars["id"]

	if err := httputils.ReadJSON(r, &updates); err != nil {
		return err
	}
	if token, err := dr.DeviceManager.Approve(id, updates); err != nil {
		return err
	} else {
		return httputils.WriteJSON(w, http.StatusOK, types.Token{Token: token})
	}
}

func (dr *devicesRouter) reject(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	if err := dr.DeviceManager.Reject(vars["id"]); err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}
