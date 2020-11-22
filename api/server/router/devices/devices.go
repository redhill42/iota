package devices

import (
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/redhill42/iota/agent"
	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/api/server/router"
	"github.com/redhill42/iota/api/types"
)

const devicePath = "/devices/{id:[^/]+}"
const claimPath = "/claims/{id:[^/]+}"

type devicesRouter struct {
	*agent.Agent
	routes []router.Route
}

func NewRouter(agent *agent.Agent) router.Router {
	r := &devicesRouter{Agent: agent}
	r.routes = []router.Route{
		router.NewGetRoute("/devices", r.list),
		router.NewPostRoute("/devices", r.create),
		router.NewGetRoute(devicePath, r.read),
		router.NewPutRoute(devicePath, r.update),
		router.NewDeleteRoute(devicePath, r.delete),
		router.NewPostRoute(devicePath+"/rpc", r.rpc),

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
	result, err := dr.DeviceManager.FindAll()
	if err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, result)
}

func (dr *devicesRouter) create(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var (
		req       map[string]interface{}
		id, token string
		ok        bool
		err       error
	)

	if err = httputils.ReadJSON(r, &req); err != nil {
		return err
	}
	if id, ok = req["id"].(string); !ok {
		return httputils.NewStatusError(http.StatusBadRequest, errors.New("Missing \"id\" attribute"))
	}
	if !validateDeviceId(id) {
		return httputils.NewStatusError(http.StatusBadRequest, errors.New("Invalid device id"))
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

var validDeviceIdPattern = regexp.MustCompile("^[a-zA-Z0-9_.@-]+$")

func validateDeviceId(id string) bool {
	return validDeviceIdPattern.MatchString(id)
}

func (dr *devicesRouter) read(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	info := make(map[string]interface{})
	if err := dr.DeviceManager.Find(vars["id"], info); err != nil {
		return err
	}

	keys := r.FormValue("keys")
	if keys != "" {
		filteredInfo := make(map[string]interface{})
		for _, key := range strings.Split(keys, ",") {
			if val, ok := info[key]; ok {
				filteredInfo[key] = val
			}
		}
		info = filteredInfo
	}

	return httputils.WriteJSON(w, http.StatusOK, &info)
}

func (dr *devicesRouter) update(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var (
		req map[string]interface{}
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

	err = dr.DeviceManager.RPC(vars["id"], r.FormValue("requestId"), req)
	if err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
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
		req map[string]interface{}
		id  string
		ok  bool
		err error
	)

	if err = httputils.ReadJSON(r, &req); err != nil {
		return err
	}
	if id, ok = req["claim-id"].(string); !ok {
		return httputils.NewStatusError(http.StatusBadRequest, errors.New("Missing \"claim-id\" attribute"))
	}
	if !validateDeviceId(id) {
		return httputils.NewStatusError(http.StatusBadRequest, errors.New("Invalid device id"))
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
	var updates map[string]interface{}
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
