package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/thkukuk/mqtt-exporter/pkg/logger"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	"gopkg.in/yaml.v3"
	"github.com/spf13/cobra"
        "github.com/influxdata/influxdb-client-go/v2"
)

const (
	defInfluxDBdatabase = "my-bucket"
)

var (
        Version = "unreleased"
        quiet   = false
        verbose = false
	configFile = "config.yaml"
        Config ConfigType
        db influxdb2.Client
)

type ConfigType struct {
	Verbose         *bool             `yaml:"verbose,omitempty"`
	Metrics         string            `yaml:"metrics"`
	Measurement     string            `yaml:"measurement"`
	Timestamp       *string           `yaml:"timestamp,omitempty"`
	Interval        time.Duration     `yaml:"interval"`
	AvoidDuplicate  bool              `yaml:"avoid_duplicate"`
	ConstantTags    map[string]string `yaml:"const_tags"`
	InfluxDB        *InfluxDBConfig   `yaml:"influxdb,omitempty"`
}

func read_yaml_config(conffile string) (ConfigType, error) {

        var config ConfigType

        file, err := ioutil.ReadFile(conffile)
        if err != nil {
                return config, fmt.Errorf("Cannot read %q: %v", conffile, err)
        }
        err = yaml.Unmarshal(file, &config)
        if err != nil {
                return config, fmt.Errorf("Unmarshal error: %v", err)
        }

        return config, nil
}


func parseMF(url string) (map[string]*dto.MetricFamily, error) {

	var reader io.Reader
	var err error

	if strings.HasPrefix(url, "http") {
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("get %q: %v", err)
		}
		defer resp.Body.Close()
		reader = resp.Body
	} else {
		reader, err = os.Open(url)
		if err != nil {
			return nil, err
		}
	}

    var parser expfmt.TextParser
    mf, err := parser.TextToMetricFamilies(reader)
    if err != nil {
        return nil, err
    }
    return mf, nil
}

func scrapAndSave(config ConfigType, old_timestamp time.Time) (time.Time) {
	mf, err := parseMF(config.Metrics)
	if err != nil {
		log.Errorf("Could not load metrics %q: %v",
			config.Metrics, err)
		return old_timestamp
	}

	var tags = make(map[string]string, len(config.ConstantTags))
	var field = make(map[string]interface{}, len(mf))

	for v, k := range config.ConstantTags {
		tags[k] = v
	}

	for k, v := range mf {
		m := v.GetMetric()
		for i := range m {
			key := k
			labels := m[i].GetLabel()
			for l := range labels {
				key = fmt.Sprintf("%s_%s:%s", key, *labels[l].Name, *labels[l].Value)
			}
			field[key] = *m[i].GetGauge().Value
		}
	}

	timestamp := time.Now()
	if config.Timestamp != nil && len(*config.Timestamp) > 0 {
		t, ok := field[*config.Timestamp].(float64)
		if !ok {
			log.Errorf("Unknown format for timestamp %v",
				field[*config.Timestamp])
		} else {
			timestamp  = time.Unix(int64(t), 0)
		}
	}

	if config.AvoidDuplicate && !timestamp.After(old_timestamp) {
		if verbose {
			log.Debugf("Skipped, %v is not newer than %v",
				timestamp, old_timestamp)
		}
		return old_timestamp
	}

	if verbose {
		log.Debugf("WriteEntry(%s, %v, %v, %v)",
			config.Measurement, tags, field, timestamp)
	}

	err = WriteEntry(db, *config.InfluxDB, config.Measurement,
		tags, field, timestamp)
	if err != nil {
		log.Errorf("Writing to db %q failed: %v",
			config.InfluxDB.Database, err)
		return old_timestamp
	}
	return timestamp
}

func main() {
        cmd := &cobra.Command{
                Use:   "prom_metrics2influxdb",
                Short: "Parses prometheus metrics and stores them in InfluxDB",
                Long: `This daemon parses metrics in prometheus format and stores the result in an InfluxDB timeseries database.`,
                Run: runCmd,
                Args:  cobra.ExactArgs(0),
        }

        cmd.Version = Version

        cmd.Flags().StringVarP(&configFile, "config", "c", configFile, "configuration file")

        cmd.Flags().BoolVarP(&quiet, "quiet", "q", quiet, "don't print any informative messages")
        cmd.Flags().BoolVarP(&verbose, "verbose", "v", verbose, "become really verbose in printing messages")

        if err := cmd.Execute(); err != nil {
                os.Exit(1)
        }
}

func runCmd(cmd *cobra.Command, args []string) {

	if !quiet {
		log.Infof("Prometheus Metrics to InfluxDB %s is starting...",
			Version)
                log.Infof("Read yaml config %q\n", configFile)
        }
        Config, err := read_yaml_config(configFile)
        if err != nil {
                log.Fatalf("Could not load config: %v", err)
        }

        if Config.Verbose != nil {
                verbose = *Config.Verbose
        }

	if Config.InfluxDB != nil {
                if len(Config.InfluxDB.Database) == 0 {
                        Config.InfluxDB.Database = defInfluxDBdatabase
                }
                db, err = ConnectInfluxDB(Config.InfluxDB)
                if err != nil {
                        log.Fatalf("Cannot connect to InfluxDB: %v", err)
                }
        } else {
                log.Fatal("No InfluxDB server specified!")
        }

	if Config.Interval == 0 {
		Config.Interval, _ = time.ParseDuration("1h")
	}

	var old_timestamp time.Time
	for {
		old_timestamp = scrapAndSave(Config, old_timestamp)
		if verbose {
			log.Debugf("sleep %s", Config.Interval)
		}
		time.Sleep(Config.Interval)
	}
}
