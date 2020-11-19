package devices

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gorilla/mux"
	"github.com/redhill42/iota/api/server"
	"github.com/redhill42/iota/api/server/httputils"
	"github.com/redhill42/iota/device"
)

func TestDevicesRouter(t *testing.T) {
	os.Setenv("IOTA_DEVICEDB_URL", "mongodb://127.0.0.1:27017/devices_test")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Devices Router Suite")
}

type fakeWriter struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func (w *fakeWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *fakeWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func (w *fakeWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *fakeWriter) Err(id string) error {
	switch w.statusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return device.DeviceNotFoundError(id)
	case http.StatusConflict:
		return device.DuplicateDeviceError(id)
	default:
		return httputils.NewStatusError(w.statusCode, nil)
	}
}

var _ = Describe("DevicesRouter", func() {
	var mgr *device.DeviceManager
	var mux *mux.Router

	BeforeEach(func() {
		var err error

		mgr, err = device.NewDeviceManager()
		Expect(err).NotTo(HaveOccurred())

		srv := server.New("")
		srv.InitRouter(NewRouter(mgr))
		mux = srv.Mux
	})

	AfterEach(func() {
		mgr.Close()
	})

	makeRequest := func(method, path, id string, req interface{}, res interface{}) error {
		var r *http.Request
		var err error

		if req != nil {
			var reqBody bytes.Buffer
			err = json.NewEncoder(&reqBody).Encode(req)
			if err != nil {
				return err
			}
			r, err = http.NewRequest(method, path, &reqBody)
			if err == nil {
				r.Header.Set("Content-Type", "application/json")
			}
		} else {
			r, err = http.NewRequest(method, path, nil)
		}
		if err != nil {
			return err
		}

		w := fakeWriter{}
		mux.ServeHTTP(&w, r)

		if err = w.Err(id); err == nil && res != nil {
			err = json.NewDecoder(&w.body).Decode(res)
		}
		return err
	}

	createDevice := func(id string, attributes map[string]interface{}) (string, error) {
		req := make(map[string]interface{})
		for k, v := range attributes {
			req[k] = v
		}
		if id != "" {
			req["id"] = id
		}

		res := struct {
			Token string `json:"token"`
		}{}

		err := makeRequest("POST", "/devices/", id, req, &res)
		return res.Token, err
	}

	getDevice := func(id string) (res map[string]interface{}, err error) {
		err = makeRequest("GET", "/devices/"+id, id, nil, &res)
		return
	}

	updateDevice := func(id string, updates map[string]interface{}) error {
		return makeRequest("POST", "/devices/"+id, id, updates, nil)
	}

	deleteDevice := func(id string) error {
		return makeRequest("DELETE", "/devices/"+id, id, nil, nil)
	}

	Describe("Create device", func() {
		It("should success with correct arguments", func() {
			_, err := createDevice("create-device-test", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail without 'id' attribute", func() {
			_, err := createDevice("", nil)
			Expect(err).To(HaveOccurred())
		})

		It("should fail with duplicate id", func() {
			deviceId := "create-device-test-dup"
			_, err := createDevice(deviceId, nil)
			Expect(err).NotTo(HaveOccurred())
			_, err = createDevice(deviceId, nil)
			Expect(err).To(MatchError(device.DuplicateDeviceError(deviceId)))
		})
	})

	Describe("Get device", func() {
		It("should success for existing device", func() {
			deviceId := "get-device-test"
			attributes := map[string]interface{}{
				"att1": "value1",
				"att2": "value2",
				"att3": "value3",
			}

			token, err := createDevice(deviceId, attributes)
			Expect(err).NotTo(HaveOccurred())

			info, err := getDevice(deviceId)
			Expect(err).NotTo(HaveOccurred())
			Expect(info["id"]).To(Equal(deviceId))
			Expect(info["token"]).To(Equal(token))
			Expect(info["att1"]).To(Equal("value1"))
			Expect(info["att2"]).To(Equal("value2"))
			Expect(info["att3"]).To(Equal("value3"))
		})

		It("should fail if device not found", func() {
			deviceId := "get-device-test-not-found"
			_, err := getDevice(deviceId)
			Expect(err).To(MatchError(device.DeviceNotFoundError(deviceId)))
		})
	})

	Describe("Update device", func() {
		It("should success for existing device", func() {
			deviceId := "update-device-test"
			attributes := map[string]interface{}{
				"att1": "value1",
				"att2": "value2",
				"att3": "value3",
			}

			_, err := createDevice(deviceId, attributes)
			Expect(err).NotTo(HaveOccurred())

			updates := map[string]interface{}{
				"att2": "new value2",
				"att4": "new value4",
			}
			err = updateDevice(deviceId, updates)
			Expect(err).NotTo(HaveOccurred())

			info, err := getDevice(deviceId)
			Expect(err).NotTo(HaveOccurred())
			Expect(info["att1"]).To(Equal("value1"))
			Expect(info["att2"]).To(Equal("new value2"))
			Expect(info["att3"]).To(Equal("value3"))
			Expect(info["att4"]).To(Equal("new value4"))
		})

		It("should success if no fields updated", func() {
			deviceId := "update-device-test-noop"
			_, err := createDevice(deviceId, nil)
			Expect(err).NotTo(HaveOccurred())

			err = updateDevice(deviceId, map[string]interface{}{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail if device not found", func() {
			deviceId := "update-device-test-not-found"
			updates := map[string]interface{}{"att1": "value1"}
			err := updateDevice(deviceId, updates)
			Expect(err).To(MatchError(device.DeviceNotFoundError(deviceId)))
		})
	})

	Describe("Delete device", func() {
		It("should success for existing device", func() {
			deviceId := "delete-device-test"
			_, err := createDevice(deviceId, nil)
			Expect(err).NotTo(HaveOccurred())

			err = deleteDevice(deviceId)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail if device not found", func() {
			deviceId := "delete-device-test-not-found"
			err := deleteDevice(deviceId)
			Expect(err).To(MatchError(device.DeviceNotFoundError(deviceId)))
		})

		It("should not find device after deletion", func() {
			deviceId := "delete-device-test-removed"
			_, err := createDevice(deviceId, nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = getDevice(deviceId)
			Expect(err).NotTo(HaveOccurred())

			err = deleteDevice(deviceId)
			Expect(err).NotTo(HaveOccurred())

			_, err = getDevice(deviceId)
			Expect(err).To(MatchError(device.DeviceNotFoundError(deviceId)))
		})
	})
})
