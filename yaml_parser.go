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
// A resource defined in a puppet manifest.
//
type Resource struct {
	Name string
	File string
	Line string
}

//
// Define a structure for our results.
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
	At_Unix int64

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
	Log_Messages []string

	//
	// Resources which have failed/changed/been skipped.
	//
	// These include the file/line in which they were defined
	// in the puppet manifest(s).
	//
	Resources_Failed  []Resource
	Resources_Changed []Resource
	Resources_Skipped []Resource
}

//
// Parse the given content into a struct, which we return.
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
	x.At_Unix = t.Unix()
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
	t_r, _ := regexp.Compile("Total ([0-9.]+)")
	f_r, _ := regexp.Compile("Failed ([0-9.]+)")
	s_r, _ := regexp.Compile("Skipped ([0-9.]+)")
	c_r, _ := regexp.Compile("Changed ([0-9.]+)")

	//
	// HORRID: Help me, I'm in hell.
	//
	// TODO: Improve via reflection as per log-handling.
	//
	for _, value := range resources {
		m_r := t_r.FindStringSubmatch(fmt.Sprint(value))
		if len(m_r) == 2 {
			x.Total = m_r[1]
		}
		m_f := f_r.FindStringSubmatch(fmt.Sprint(value))
		if len(m_f) == 2 {
			x.Failed = m_f[1]
		}
		m_s := s_r.FindStringSubmatch(fmt.Sprint(value))
		if len(m_s) == 2 {
			x.Skipped = m_s[1]
		}
		m_c := c_r.FindStringSubmatch(fmt.Sprint(value))
		if len(m_c) == 2 {
			x.Changed = m_c[1]
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
			x.Log_Messages = append(x.Log_Messages, m["message"])
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
			x.Resources_Skipped = append(x.Resources_Skipped,
				Resource{Name: m["title"],
					File: m["file"],
					Line: m["line"]})
		}

		// Now we should be able to look for skipped ones.
		if m["changed"] == "true" {
			x.Resources_Changed = append(x.Resources_Changed,
				Resource{Name: m["title"],
					File: m["file"],
					Line: m["line"]})
		}

		// Now we should be able to look for skipped ones.
		if m["failed"] == "true" {
			x.Resources_Failed = append(x.Resources_Failed,
				Resource{Name: m["title"],
					File: m["file"],
					Line: m["line"]})
		}
	}

	return x, nil
}
