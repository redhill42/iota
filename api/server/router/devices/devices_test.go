package devices

import (
	"net/http"
	"testing"
	"os"
	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/redhill42/iota/device"
)

func TestDevicesRouter(t *testing.T) {
	os.Setenv("IOTA_DEVICEDB_URL", "mongodb://127.0.0.1:27017/devices_test")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Devices Router Suite")
}

type FakeWriter struct {
	header http.Header
	body bytes.Buffer
}

func (w *FakeWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *FakeWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func (w *FakeWriter) WriteHeader(statusCode int) {
	// don't actually write
}

var _ = Describe("DevicesRouter", func() {
	var manager *device.DeviceManager
	var router *devicesRouter

	BeforeEach(func() {
		var err error

		manager, err = device.NewDeviceManager()
		Expect(err).NotTo(HaveOccurred())
		router = NewRouter(manager).(*devicesRouter)
	})

	AfterEach(func() {
		manager.Close()
	})

	createDevice := func(id string, attributes map[string]interface{}) (string, error) {
		req := make(map[string]interface{})
		for k, v := range attributes {
			req[k] = v
		}
		if (id != "") {
			req["id"] = id
		}

		var reqBody bytes.Buffer
		err := json.NewEncoder(&reqBody).Encode(req)
		if err != nil {
			return "", err
		}

		r, err := http.NewRequest("POST", "/devices/", &reqBody)
		if err != nil {
			return "", err
		}
		r.Header.Set("Content-Type", "application/json")

		w := FakeWriter{}
		err = router.create(&w, r, nil)
		if err != nil {
			return "", err
		}

		var res map[string]string
		err = json.NewDecoder(&w.body).Decode(&res)
		if err != nil {
			return "", err
		}

		token, ok := res["token"]
		Expect(ok).To(BeTrue())
		return token, nil
	}

	getDevice := func(id string) (map[string]interface{}, error) {
		r, err := http.NewRequest("GET", "/devices/"+id, nil)
		if err != nil {
			return nil, err
		}

		w := FakeWriter{}
		err = router.read(&w, r, map[string]string{"id": id})
		if err != nil {
			return nil, err
		}

		var res map[string]interface{}
		err = json.NewDecoder(&w.body).Decode(&res)
		return res, err
	}

	updateDevice := func(id string, updates map[string]interface{}) error {
		var reqBody bytes.Buffer
		err := json.NewEncoder(&reqBody).Encode(updates)
		if err != nil {
			return err
		}

		r, err := http.NewRequest("POST", "/devices/"+id, &reqBody)
		if err != nil {
			return err
		}
		r.Header.Set("Content-Type", "application/json")

		w := FakeWriter{}
		return router.update(&w, r, map[string]string{"id": id})
	}

	deleteDevice := func(id string) error {
		r, err := http.NewRequest("DELETE", "/devices/"+id, nil)
		if err != nil {
			return err
		}

		w := FakeWriter{}
		return router.delete(&w, r, map[string]string{"id": id})
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
			attributes := map[string]interface{} {
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
