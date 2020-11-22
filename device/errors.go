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

// The DuplicateClaimError indicates that a device claiming is already exists.
type DuplicateClaimError string

// The ClaimNotFoundError indicates that a device claiming is not found.
type ClaimNotFoundError string

func (e DuplicateClaimError) Error() string {
	return fmt.Sprintf("Device claim with id '%s' is in progress, please wait.", string(e))
}

func (e DuplicateClaimError) HTTPErrorStatusCode() int {
	return http.StatusConflict
}

func (e ClaimNotFoundError) Error() string {
	return fmt.Sprintf("No such device claim: %s", string(e))
}

func (e ClaimNotFoundError) HTTPErrorStatusCode() int {
	return http.StatusNotFound
}
