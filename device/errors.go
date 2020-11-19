package device

import (
	"fmt"
	"net/http"
)

// The DuplicateDeviceError indicates that a device already exists in the database
// when creating device.
type DuplicateDeviceError string

// The DeviceNotFoundError indicates that a device not found in the database.
type DeviceNotFoundError string

func (e DuplicateDeviceError) Error() string {
	return fmt.Sprintf("Device already exists: %s", string(e))
}

func (e DuplicateDeviceError) HTTPErrorStatusCode() int {
	return http.StatusConflict
}

func (e DeviceNotFoundError) Error() string {
	return fmt.Sprintf("Device not found: %s", string(e))
}

func (e DeviceNotFoundError) HTTPErrorStatusCode() int {
	return http.StatusNotFound
}
