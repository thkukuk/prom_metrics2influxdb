package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

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
        logger  = log.New(os.Stdout, "", log.LstdFlags)
        logerr  = log.New(os.Stderr, "", log.LstdFlags)
	configFile = "config.yaml"
        Config ConfigType
        db influxdb2.Client
)

type ConfigType struct {
	Metrics         string            `yaml:"metrics"`
	Measurement     string            `yaml:"measurement"`
	Timestamp       string            `yaml:"timestamp,omitempty"`
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
		logger.Printf("Prometheus Metrics to InfluxDB %s is starting...",
			Version)
                logger.Printf("Read yaml config %q\n", configFile)
        }
        Config, err := read_yaml_config(configFile)
        if err != nil {
                logerr.Fatalf("Could not load config: %v", err)
        }

	if Config.InfluxDB != nil {
                if len(Config.InfluxDB.Database) == 0 {
                        Config.InfluxDB.Database = defInfluxDBdatabase
                }
                db, err = ConnectInfluxDB(Config.InfluxDB)
                if err != nil {
                        logerr.Fatalf("Cannot connect to InfluxDB: %v", err)
                }
        } else {
                logger.Fatal("No InfluxDB server specified!")
        }

	if Config.Interval == 0 {
		Config.Interval, _ = time.ParseDuration("1h")
	}

	var old_timestamp time.Time
	for {
		mf, err := parseMF(Config.Metrics)
		if err != nil {
			logerr.Fatalf("Could not load metrics %q: %v",
				Config.Metrics, err)
		}

		var tags = make(map[string]string, len(Config.ConstantTags))
		var field = make(map[string]interface{}, len(mf))

		for v, k := range Config.ConstantTags {
			tags[k] = v
		}

		for k, v := range mf {
			m := v.GetMetric()
			// XXX m[0] -> can we have more/different index?
			field[k] = *m[0].GetGauge().Value
		}

		timestamp := time.Now()
		if len(Config.Timestamp) > 0 {
			t, ok := field[Config.Timestamp].(float64)
			if !ok {
				logerr.Printf("What is timestamp? %v",
					field[Config.Timestamp])
			}
			timestamp  = time.Unix(int64(t), 0)
		}

		if Config.AvoidDuplicate && !timestamp.After(old_timestamp) {
			if verbose {
				logger.Printf("Skipped, %v is not newer than %v",
					timestamp, old_timestamp)
			}
		} else {
			if verbose {
				logger.Printf("WriteEntry(%s, %v, %v, %v)",
					Config.Measurement, tags, field, timestamp)
			}

			err = WriteEntry(db, *Config.InfluxDB, Config.Measurement,
				tags, field, timestamp)
			if err != nil {
				logerr.Printf("Writing to db %q failed: %v",
					Config.InfluxDB.Database, err)
			}

			old_timestamp = timestamp
		}

		if verbose {
			logger.Printf("sleep %s", Config.Interval)
		}
		time.Sleep(Config.Interval)
	}
}
