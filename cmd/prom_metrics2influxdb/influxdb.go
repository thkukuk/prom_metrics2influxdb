package main

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/thkukuk/prom_metrics2influxdb/pkg/logger"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/influxdata/influxdb-client-go/v2"
)

const (
	defInfluxDBPort = "8086"
)

type InfluxDBConfig struct {
	Server       string `yaml:"server"`
	Port         string `yaml:"port"`
	Database     string `yaml:"database"`
	Organization string `yaml:"organization"`
	Token        string `yaml:"token,omitempty"`
}

func WriteEntry(client influxdb2.Client, config InfluxDBConfig, measurement string, tag map[string]string, field map[string]interface{}, timestamp time.Time) error {
	writeAPI := client.WriteAPIBlocking(config.Organization, config.Database)

	p := influxdb2.NewPoint(measurement, tag, field, timestamp)
	// write asynchronously
	return writeAPI.WritePoint(context.Background(), p)
}

func createDatabase(client influxdb2.Client, config *InfluxDBConfig) error {
	if verbose {
		log.Debug("Check if the database needs to be created...")
	}
	ctx := context.Background()

	bucket, err := client.BucketsAPI().FindBucketByName(ctx, config.Database)
	if verbose && err != nil {
		log.Debugf("Error finding bucket %q: %v", config.Database, err)
	}
	// so we found the database
	if bucket != nil {
		return nil
	}

	// Get organization that will own new bucket
	org, err := client.OrganizationsAPI().FindOrganizationByName(ctx, config.Organization)
	if err != nil {
		return fmt.Errorf("Error finding organization %q: %v",
			config.Organization, err)
	}
	// Create  a bucket with 1 day retention policy
	_, err = client.BucketsAPI().CreateBucketWithName(ctx, org,
		config.Database, domain.RetentionRule{EverySeconds: 0})
	if err != nil {
		return fmt.Errorf("Error crating bucket %q: %v",
			config.Database, err)
	}

	if !quiet {
		log.Infof("Created database %q in organization %q\n", config.Database, config.Organization)
	}
	return nil
}

func ConnectInfluxDB(config *InfluxDBConfig) (influxdb2.Client, error) {

	token := os.Getenv("INFLUXDB_TOKEN")
        if token != "" {
                config.Token = token
        }

	// Create a new client using an InfluxDB server base URL and an
	// authentication token
	if len(config.Port) == 0 {
		config.Port = defInfluxDBPort
	}
	serverUrl := fmt.Sprintf("http://%s:%s",
		config.Server, config.Port)
	client := influxdb2.NewClient(serverUrl, config.Token)
	defer client.Close()

	health, err := client.Health(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Cannot get health status: %v", err)
	} else if health.Status == domain.HealthCheckStatusFail {
		return nil, fmt.Errorf("Database not healthy: %v", health)
	}

	err = createDatabase(client, config)
	if err != nil {
		log.Warnf("Cannot verify database, maybe InfluxDB v1 is used? Please make sure it exists.")
		if verbose {
			log.Debug(err)
		}
	}

	return client, nil
}
