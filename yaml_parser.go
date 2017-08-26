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
//  Here we have some simple methods that each parse a part of the
// YAML file, updating the structure they are passed.
//
//  These snippets are broken down to avoid an uber-complex
// set of code in the ParsePuppetReport method.
//

//
// parseHost reads the `host` parameter from the YAML and populates
// the given report-structure with suitable values.
//
func parseHost(y *simpleyaml.Yaml, out *PuppetReport) error {
	//
	// Get the hostname.
	//
	host, err := y.Get("host").String()
	if err != nil {
		return errors.New("Failed to get 'host' from YAML")
	}

	//
	// Ensure the hostname passes a simple regexp
	//
	reg, _ := regexp.Compile("^([a-z0-9._-]+)$")
	if !reg.MatchString(host) {
		return errors.New("The submitted 'host' field failed our security check")
	}

	out.Fqdn = host
	return nil
}

//
// parseTime reads the `time` parameter from the YAML and populates
// the given report-structure with suitable values.
//
func parseTime(y *simpleyaml.Yaml, out *PuppetReport) error {

	//
	// Get the time puppet executed
	//
	at, err := y.Get("time").String()
	if err != nil {
		return errors.New("Failed to get 'time' from YAML")
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
		return errors.New("Failed to parse 'time' from YAML")
	}

	// update the struct
	out.AtUnix = t.Unix()
	out.At = at

	return nil
}

//
// parseStatus reads the `status` parameter from the YAML and populates
// the given report-structure with suitable values.
//
func parseStatus(y *simpleyaml.Yaml, out *PuppetReport) error {
	//
	// Get the status
	//
	state, err := y.Get("status").String()
	if err != nil {
		return errors.New("Failed to get 'status' from YAML")
	}

	switch state {
	case "changed":
	case "unchanged":
	case "failed":
	default:
		return errors.New("Unexpected 'status' - " + state)
	}

	out.State = state
	return nil
}

//
// parseRuntime reads the `metrics.time.values` parameters from the YAML
// and populates given report-structure with suitable values.
//
func parseRuntime(y *simpleyaml.Yaml, out *PuppetReport) error {

	//
	// Get the run-time this execution took.
	//
	times, err := y.Get("metrics").Get("time").Get("values").Array()
	if err != nil {
		return err
	}

	r, _ := regexp.Compile("Total ([0-9.]+)")

	runtime := ""

	//
	// HORRID: Help me, I'm in hell.
	//
	// TODO: Improve via reflection as per log-handling.
	//
	for _, value := range times {
		match := r.FindStringSubmatch(fmt.Sprint(value))
		if len(match) == 2 {
			runtime = match[1]
		}
	}
	out.Runtime = runtime
	return nil
}

//
// parseResources looks for the counts of resources which have been
// failed, changed, skipped, etc, and updates the given report-structure
// with those values.
//
func parseResources(y *simpleyaml.Yaml, out *PuppetReport) error {

	resources, err := y.Get("metrics").Get("resources").Get("values").Array()
	if err != nil {
		return err
	}

	tr, _ := regexp.Compile("Total ([0-9.]+)")
	fr, _ := regexp.Compile("Failed ([0-9.]+)")
	sr, _ := regexp.Compile("Skipped ([0-9.]+)")
	cr, _ := regexp.Compile("Changed ([0-9.]+)")

	total := ""
	changed := ""
	failed := ""
	skipped := ""

	//
	// HORRID: Help me, I'm in hell.
	//
	// TODO: Improve via reflection as per log-handling.
	//
	for _, value := range resources {
		mr := tr.FindStringSubmatch(fmt.Sprint(value))
		if len(mr) == 2 {
			total = mr[1]
		}
		mf := fr.FindStringSubmatch(fmt.Sprint(value))
		if len(mf) == 2 {
			failed = mf[1]
		}
		ms := sr.FindStringSubmatch(fmt.Sprint(value))
		if len(ms) == 2 {
			skipped = ms[1]
		}
		mc := cr.FindStringSubmatch(fmt.Sprint(value))
		if len(mc) == 2 {
			changed = mc[1]
		}
	}

	out.Total = total
	out.Changed = changed
	out.Failed = failed
	out.Skipped = skipped
	return nil
}

//
// parseLogs updates the given report with any logged messages.
//
func parseLogs(y *simpleyaml.Yaml, out *PuppetReport) error {
	logs, err := y.Get("logs").Array()
	if err != nil {
		return errors.New("Failed to get 'logs' from YAML")
	}

	var logged []string

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
			logged = append(logged, m["message"])
		}
	}

	out.LogMessages = logged
	return nil
}

//
// parseResults updates the given report with details of any resource
// which was failed, changed, or skipped.
//
func parseResults(y *simpleyaml.Yaml, out *PuppetReport) error {
	rs, err := y.Get("resource_statuses").Map()
	if err != nil {
		return errors.New("Failed to get 'resource_statuses' from YAML")
	}

	var failed []Resource
	var changed []Resource
	var skipped []Resource

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
			skipped = append(skipped,
				Resource{Name: m["title"],
					File: m["file"],
					Line: m["line"]})
		}

		// Now we should be able to look for skipped ones.
		if m["changed"] == "true" {
			changed = append(changed,
				Resource{Name: m["title"],
					File: m["file"],
					Line: m["line"]})
		}

		// Now we should be able to look for skipped ones.
		if m["failed"] == "true" {
			failed = append(failed,
				Resource{Name: m["title"],
					File: m["file"],
					Line: m["line"]})
		}
	}

	out.ResourcesSkipped = skipped
	out.ResourcesFailed = failed
	out.ResourcesChanged = changed

	return nil

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
	// Parse the hostname
	//
	hostError := parseHost(yaml, &x)
	if hostError != nil {
		return x, hostError
	}

	//
	// Parse the time.
	//
	timeError := parseTime(yaml, &x)
	if timeError != nil {
		return x, timeError
	}

	//
	// Parse the status
	//
	stateError := parseStatus(yaml, &x)
	if stateError != nil {
		return x, stateError
	}

	//
	// Parse the runtime of this execution
	//
	runError := parseRuntime(yaml, &x)
	if runError != nil {
		return x, runError
	}

	//
	// Get the resource-data from this run
	//
	resourcesError := parseResources(yaml, &x)
	if resourcesError != nil {
		return x, resourcesError
	}

	//
	// Get the logs from this run
	//
	logsError := parseLogs(yaml, &x)
	if logsError != nil {
		return x, logsError
	}

	//
	// Finally the resources
	//
	resError := parseResults(yaml, &x)
	if resError != nil {
		return x, resError
	}

	return x, nil
}
