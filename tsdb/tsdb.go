package tsdb

import (
	"context"
	"errors"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redhill42/iota/config"
	"github.com/sirupsen/logrus"
)

// TSDB is an interface to time series database
type TSDB interface {
	// WriteRecord writes asynchronously measurement record into database.
	WriteRecord(record string)
	// Close ensures all ongoing asynchronous write client finish.
	Close()
}

type influx struct {
	client   influxdb.Client
	writeAPI api.WriteAPI
}

func New() (TSDB, error) {
	serverURL := config.GetOption("influxdb", "server")
	token := config.GetOption("influxdb", "token")
	org := config.GetOption("influxdb", "org")
	bucket := config.GetOption("influxdb", "bucket")

	if serverURL == "" || token == "" {
		return nil, errors.New("InfluxDB was not configured correctly")
	}
	if org == "" {
		org = "iota"
	}
	if bucket == "" {
		bucket = "iota"
	}

	client := influxdb.NewClient(serverURL, token)
	if _, err := client.Ready(context.Background()); err != nil {
		return nil, err
	}

	writeAPI := client.WriteAPI(org, bucket)
	go reportErrors(writeAPI.Errors())
	return &influx{client, writeAPI}, nil
}

func (db *influx) WriteRecord(record string) {
	db.writeAPI.WriteRecord(record)
}

func (db *influx) Close() {
	db.client.Close()
}

func reportErrors(errCh <-chan error) {
	for err := range errCh {
		logrus.Error(err)
	}
}
