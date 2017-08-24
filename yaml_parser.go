//
// This package is the sole place that we extract data from the YAML
// that Puppet submits to us.
//
// Here is where we're going to extract:
//
//  * Logged messages
//  * Runtime
//  * etc.
//

package main

import (
	"errors"
	"fmt"
	"github.com/smallfish/simpleyaml"
	"reflect"
	"regexp"
	"strings"
	"time"
)

//
// Resource refers to a resource in your puppet modules, a resource has
// a name, along with the file & line-number it was defined in within your
// manifest
//
type Resource struct {
	Name string
	File string
	Line string
}

//
// PuppetReport stores the details of a single run of puppet.
//
type PuppetReport struct {

	//
	// FQDN of the node.
	//
	Fqdn string

	//
	// State of the run: changed unchanged, etc.
	//
	State string

	//
	// The time the puppet-run was completed.
	//
	At string

	//
	// The time the puppet-run was completed, as seconds past epoch
	//
	AtUnix int64

	//
	// The time puppet took to run, in seconds.
	//
	Runtime string

	//
	// Resources now.  These are counts.
	//
	// NOTE: These are actually numbers.
	//
	Failed  string
	Changed string
	Total   string
	Skipped string

	//
	// Logs messages.
	//
	LogMessages []string

	//
	// Resources which have failed/changed/been skipped.
	//
	// These include the file/line in which they were defined
	// in the puppet manifest(s), due to their use of the Resource
	// structure
	//
	ResourcesFailed  []Resource
	ResourcesChanged []Resource
	ResourcesSkipped []Resource
}

//
// ParsePuppetReport is our main function in this module.  Given an
// array of bytes we read the input and produce a PuppetReport structure.
//
// Various (simple) error conditions are handled to ensure that the result
// is somewhat safe - for example we must have some fields such as
// `hostname`, `time`, etc.
//
func ParsePuppetReport(content []byte) (PuppetReport, error) {
	//
	// The return-value.
	//
	var x PuppetReport

	//
	// Parse the YAML.
	//
	yaml, err := simpleyaml.NewYaml(content)
	if err != nil {
		return x, errors.New("Failed to parse YAML")
	}

	//
	// Get the hostname.
	//
	x.Fqdn, err = yaml.Get("host").String()
	if err != nil {
		return x, errors.New("Failed to get 'host' from YAML")
	}

	//
	// Ensure the hostname passes a simple regexp
	//
	reg, _ := regexp.Compile("^([a-z0-9._-]+)$")
	if !reg.MatchString(x.Fqdn) {
		return x, errors.New("The submitted 'host' field failed our security check")
	}

	//
	// Get the time puppet executed
	//
	at, err := yaml.Get("time").String()
	if err != nil {
		return x, errors.New("Failed to get 'time' from YAML")
	}

	// Strip any quotes that might surround the time.
	at = strings.Replace(at, "'", "", -1)

	// Convert "T" -> " "
	at = strings.Replace(at, "T", " ", -1)

	// strip the time at the first period.
	parts := strings.Split(at, ".")
	at = parts[0]
	layout := "2006-01-02 15:04:05"

	t, err := time.Parse(layout, at)
	if err != nil {
		return x, errors.New("Failed to parse 'time' from YAML")
	}

	// update the struct
	x.AtUnix = t.Unix()
	x.At = at

	//
	// Get the status
	//
	x.State, err = yaml.Get("status").String()
	if err != nil {
		return x, errors.New("Failed to get 'status' from YAML")
	}

	switch x.State {
	case "changed":
	case "unchanged":
	case "failed":
	default:
		return x, errors.New("Unexpected 'status' - " + x.State)
	}

	//
	// Get the run-time this execution took.
	//
	times, err := yaml.Get("metrics").Get("time").Get("values").Array()
	r, _ := regexp.Compile("Total ([0-9.]+)")

	//
	// HORRID: Help me, I'm in hell.
	//
	// TODO: Improve via reflection as per log-handling.
	//
	for _, value := range times {
		match := r.FindStringSubmatch(fmt.Sprint(value))
		if len(match) == 2 {
			x.Runtime = match[1]
		}
	}

	//
	// Get the resource-data from this run
	//
	resources, err := yaml.Get("metrics").Get("resources").Get("values").Array()
	tr, _ := regexp.Compile("Total ([0-9.]+)")
	fr, _ := regexp.Compile("Failed ([0-9.]+)")
	sr, _ := regexp.Compile("Skipped ([0-9.]+)")
	cr, _ := regexp.Compile("Changed ([0-9.]+)")

	//
	// HORRID: Help me, I'm in hell.
	//
	// TODO: Improve via reflection as per log-handling.
	//
	for _, value := range resources {
		mr := tr.FindStringSubmatch(fmt.Sprint(value))
		if len(mr) == 2 {
			x.Total = mr[1]
		}
		mf := fr.FindStringSubmatch(fmt.Sprint(value))
		if len(mf) == 2 {
			x.Failed = mf[1]
		}
		ms := sr.FindStringSubmatch(fmt.Sprint(value))
		if len(ms) == 2 {
			x.Skipped = ms[1]
		}
		mc := cr.FindStringSubmatch(fmt.Sprint(value))
		if len(mc) == 2 {
			x.Changed = mc[1]
		}
	}

	//
	// Try to get the values of any logged messages here.
	//
	//    https://stackoverflow.com/questions/38185916/convert-interface-to-map-in-golang
	//
	logs, err := yaml.Get("logs").Array()
	if err != nil {
		return x, errors.New("Failed to get 'logs' from YAML")
	}

	for _, v2 := range logs {

		// create a map
		m := make(map[string]string)

		v := reflect.ValueOf(v2)
		if v.Kind() == reflect.Map {
			for _, key := range v.MapKeys() {
				strct := v.MapIndex(key)

				// Store the key/val in the map.
				key, val := key.Interface(), strct.Interface()
				m[key.(string)] = fmt.Sprint(val)
			}
		}

		if len(m["message"]) > 0 {
			x.LogMessages = append(x.LogMessages, m["message"])
		}
	}

	rs, err := yaml.Get("resource_statuses").Map()
	if err != nil {
		return x, errors.New("Failed to get 'resource_statuses' from YAML")
	}

	for _, v2 := range rs {

		// create a map here.
		m := make(map[string]string)

		v := reflect.ValueOf(v2)
		if v.Kind() == reflect.Map {
			for _, key := range v.MapKeys() {
				strct := v.MapIndex(key)

				// Store the key/val in the map.
				key, val := key.Interface(), strct.Interface()
				m[key.(string)] = fmt.Sprint(val)
			}
		}

		// Now we should be able to look for skipped ones.
		if m["skipped"] == "true" {
			x.ResourcesSkipped = append(x.ResourcesSkipped,
				Resource{Name: m["title"],
					File: m["file"],
					Line: m["line"]})
		}

		// Now we should be able to look for skipped ones.
		if m["changed"] == "true" {
			x.ResourcesChanged = append(x.ResourcesChanged,
				Resource{Name: m["title"],
					File: m["file"],
					Line: m["line"]})
		}

		// Now we should be able to look for skipped ones.
		if m["failed"] == "true" {
			x.ResourcesFailed = append(x.ResourcesFailed,
				Resource{Name: m["title"],
					File: m["file"],
					Line: m["line"]})
		}
	}

	return x, nil
}
