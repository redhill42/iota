package devices

import (
	"errors"
	"net/http"

	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/api/server/router"
	"github.com/redhill42/iota/device"
)

const devicePath = "/devices/{id:[^/]+}"

type devicesRouter struct {
	mgr    *device.DeviceManager
	routes []router.Route
}

func NewRouter(mgr *device.DeviceManager) router.Router {
	r := &devicesRouter{mgr: mgr}
	r.routes = []router.Route{
		router.NewGetRoute("/devices/", r.list),
		router.NewPostRoute("/devices/", r.create),
		router.NewGetRoute(devicePath, r.read),
		router.NewPostRoute(devicePath, r.update),
		router.NewDeleteRoute(devicePath, r.delete),
	}
	return r
}

func (dr *devicesRouter) Routes() []router.Route {
	return dr.routes
}

func (dr *devicesRouter) list(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	result, err := dr.mgr.FindAll()
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
	if token, err = dr.mgr.CreateToken(id); err != nil {
		return err
	}
	if err = dr.mgr.Create(id, token, req); err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
	})
}

func (dr *devicesRouter) read(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	info := make(map[string]interface{})
	if err := dr.mgr.Find(vars["id"], info); err != nil {
		return err
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
	return dr.mgr.Update(vars["id"], req)
}

func (dr *devicesRouter) delete(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	return dr.mgr.Remove(vars["id"])
}
