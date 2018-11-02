package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/maxatome/go-vitotrol"
)

func VitotrolInit(vconf *ConfigVitotrol) *vitotrol.Session {
	var err error
	for {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\nSleeping before retrying...\n", err)
			time.Sleep(time.Duration(vconf.RetryTimeout) * time.Second)
		}

		pVitotrol := &vitotrol.Session{}

		fmt.Println("Vitotrol login...")
		err = pVitotrol.Login(vconf.Login, vconf.Password)
		if err != nil {
			err = fmt.Errorf("Login failed: %s", err)
			continue
		}

		fmt.Println("Vitotrol GetDevices...")
		err = pVitotrol.GetDevices()
		if err != nil {
			err = fmt.Errorf("GetDevices failed: %s", err)
			continue
		}
		if len(pVitotrol.Devices) == 0 {
			err = fmt.Errorf("No device found")
			continue
		}
		fmt.Printf("%d device(s) found\n", len(pVitotrol.Devices))
		return pVitotrol
	}
}

func getAttrValue(vdev *vitotrol.Device, attrID vitotrol.AttrID) (value interface{}) {
	value, _ = vitotrol.AttributesRef[attrID].
		Type.Vitodata2NativeValue(vdev.Attributes[attrID].Value)

	// uint64 handled from influx 1.4
	if vuint64, ok := value.(uint64); ok {
		value = int(vuint64)
	}
	return
}

var customAttr = regexp.MustCompile(
	`^([a-zA-Z0-9]+)[-_]0x([a-fA-F0-9]{1,4})\z`)

func main() {
	configFile := flag.String("config", "", "config file")

	flag.Parse()

	if *configFile == "" {
		fmt.Fprintln(os.Stderr, "config file is missing")
		os.Exit(1)
	}
	conf, err := ReadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read config: %s\n", err)
		os.Exit(1)
	}

	// Resolve fields
	for _, cdev := range conf.Devices {
		attrs := make(map[vitotrol.AttrID]struct{}, len(cdev.Fields))

		for _, fieldName := range cdev.Fields {
			// Computed attribute
			if cattr, ok := computedAttrs[fieldName]; ok {
				for _, attrID := range cattr.Attrs {
					attrs[attrID] = struct{}{}
				}
				cdev.computedFields = append(cdev.computedFields, fieldName)
			} else {
				// Already known attribute
				attrID, ok := vitotrol.AttributesNames2IDs[fieldName]
				if !ok {
					// Custom attribute
					m := customAttr.FindStringSubmatch(fieldName)
					if m == nil {
						fmt.Fprintf(os.Stderr, "Unknown attribute `%s'\n", fieldName)
						os.Exit(1)
					}

					attrRef := vitotrol.AttrRef{
						Type:   vitotrol.TypeDouble,
						Access: vitotrol.ReadOnly,
						Name:   m[1],
					}
					tmpID, _ := strconv.ParseUint(m[2], 16, 16)
					attrID = vitotrol.AttrID(tmpID)

					vitotrol.AddAttributeRef(attrID, attrRef)
				}

				attrs[attrID] = struct{}{}
			}
		}

		if len(attrs) == 0 {
			fmt.Fprintf(os.Stderr, "No attributes for device %s/location %s\n",
				cdev.Name, cdev.Location)
			os.Exit(1)
		}

		cdev.attrs = make([]vitotrol.AttrID, 0, len(attrs))
		for attrID := range attrs {
			cdev.attrs = append(cdev.attrs, attrID)
		}
	}

	// Create a new Client
	influx, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     conf.Influx.Address,
		Username: conf.Influx.Login,
		Password: conf.Influx.Password,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Influx new client failed: %s\n", err)
		os.Exit(1)
	}

newVitotrol:
	for {
		pVitotrol := VitotrolInit(&conf.Vitotrol)

		for {
			start := time.Now()

			for _, vdev := range pVitotrol.Devices {
				if !vdev.IsConnected {
					continue
				}

				// Check if this device has a configuration
				cdev := conf.GetConfigDevice(vdev.DeviceName, vdev.LocationName)
				if cdev == nil {
					continue
				}

				ch, err := vdev.RefreshDataWait(pVitotrol, cdev.attrs)
				if err != nil {
					fmt.Fprintf(os.Stderr, "RefreshData error: %s\n", err)
					continue newVitotrol
				}

				// Create a new point batch
				bp, err := client.NewBatchPoints(client.BatchPointsConfig{
					Database:         cdev.Database,
					Precision:        cdev.Precision,
					RetentionPolicy:  cdev.RetentionPolicy,
					WriteConsistency: cdev.WriteConsistency,
				})
				if err != nil {
					fmt.Printf("influx.NewBatchPoints failed: %s\n", err)
					os.Exit(1)
				}

				if err = <-ch; err != nil {
					fmt.Fprintf(os.Stderr, "RefreshData failed: %s\n", err)
					continue newVitotrol
				}

				now := time.Now()

				err = vdev.GetData(pVitotrol, cdev.attrs)
				if err != nil {
					fmt.Fprintf(os.Stderr, "GetData error: %s\n", err)
					continue newVitotrol
				}

				fields := map[string]interface{}{}

				for _, attrID := range cdev.attrs {
					fields[vitotrol.AttributesRef[attrID].Name] =
						getAttrValue(&vdev, attrID)
				}

				// Computed attrs
				for _, fieldName := range cdev.computedFields {
					fields[fieldName] = computedAttrs[fieldName].Compute(&vdev)
				}

				// If no tags are configured, use defaults
				tags := cdev.Tags
				if len(tags) == 0 {
					tags = map[string]string{
						"device":   vdev.DeviceName,
						"location": vdev.LocationName,
					}
				}

				point, err := client.NewPoint(cdev.Measurement, tags, fields, now)
				if err != nil {
					fmt.Printf("influx.NewPoint failed: %s\n", err)
					os.Exit(1)
				}

				bp.AddPoint(point)

				fmt.Printf("%s %s → writing batch...\n",
					now.Format(time.RFC3339),
					now.Sub(start).Truncate(time.Millisecond),
				)

				// Write the batch
				err = influx.Write(bp)
				if err != nil {
					fmt.Printf("influx.Write failed: %s\n", err)
					os.Exit(1)
				}

				time.Sleep(time.Duration(conf.Vitotrol.Frequency) * time.Second)
			}
		}
	}
}
