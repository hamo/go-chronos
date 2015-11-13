package chronos

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	JOB_TYPE_DependencyBased = iota
	JOB_TYPE_ScheduleBased
	JOB_TYPE_Unknown
)

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Job struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	Command   string   `json:"command"`
	Shell     bool     `json:"shell"`
	Arguments []string `json:"arguments"`
	RunAsUser string   `json:"runAsUser"`

	EnvironmentVariables []*EnvVar `json:"environmentVariables"`

	Async                 bool `json:"async"`
	Disabled              bool `json:"disabled"`
	HighPriority          bool `json:"highPriority"`
	SoftError             bool `json:"softError"`
	DataProcessingJobType bool `json:"dataProcessingJobType"`

	Container *Container `json:"container,omitempty"`

	CPUs   float64 `json:"cpus"`
	Disk   float64 `json:"disk"`
	Memory float64 `json:"mem"`

	Uris []string `json:"uris"`

	Epsilon string `json:"epsilon"`

	SuccessCount           int    `json:"successCount"`
	ErrorCount             int    `json:"errorCount"`
	LastSuccess            string `json:"lastSuccess"`
	LastError              string `json:"lastError"`
	ErrorsSinceLastSuccess int    `json:"errorsSinceLastSuccess"`

	Executor      string `json:"executor"`
	ExecutorFlags string `json:"executorFlags"`

	Retries int `json:"retries"`

	Owner     string `json:"owner"`
	OwnerName string `json:"ownerName"`

	Constraints [][]string `json:"constraints"`

	// These 2 only appear in ScheduleBasedJob
	Schedule         string `json:"schedule,omitempty"`
	ScheduleTimeZone string `json:"scheduleTimeZone,omitempty"`

	// This one only appears in DependencyBasedJob
	Parents []string `json:"parents,omitempty"`
}

func NewJob() *Job {
	j := new(Job)
	j.Init()
	return j
}

func NewContainerJob() *Job {
	j := NewJob()
	j.Container = NewDockerContainer()
	return j
}

func (j *Job) Init() {
	j.Shell = true
	j.Epsilon = "PT60S"
	j.Retries = 2
	j.Async = false
	j.DataProcessingJobType = false
	j.Disabled = false

	j.Arguments = make([]string, 0)
	j.EnvironmentVariables = make([]*EnvVar, 0)
	j.Uris = make([]string, 0)
	j.Constraints = make([][]string, 0)
}

func (j *Job) SanityCheck() (bool, error) {
	if j.Type() == JOB_TYPE_Unknown {
		return false, errors.New("chronos: job type unknown")
	}

	switch j.Type() {
	case JOB_TYPE_ScheduleBased:
		if pass, err := j.CheckSchedule(); !pass {
			return false, err
		}
	case JOB_TYPE_DependencyBased:

	}

	return true, nil
}

func (j *Job) CheckSchedule() (bool, error) {
	arrs := strings.Split(j.Schedule, "/")
	if len(arrs) != 3 {
		return false, errors.New("chronos: schedule should contain 3 elements")
	}

	repeat := arrs[0]
	startTime := arrs[1]
	interval := arrs[2]

	repeatRegExp := regexp.MustCompile(`^R(?P<times>\d+)?$`)
	startTimeRegExp := regexp.MustCompile(`^(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})T(?P<hour>\d{2}):(?P<minute>\d{2}):(?P<second>\d{2})(?P<zone>(Z|(\+|-)\d{2}:\d{2}))$`)
	intervalRegExp := regexp.MustCompile(`^P((?P<year>\d+)Y)?((?P<month>\d+)M)?((?P<day>\d+)D)?(T((?P<hour>\d+)H)?((?P<minute>\d+)M)?((?P<second>\d+)S)?)?$`)

	if !repeatRegExp.MatchString(repeat) {
		return false, errors.New("chronos: schedule: repeat field syntax error")
	}

	if !startTimeRegExp.MatchString(startTime) {
		return false, errors.New("chronos: schedule: startTime field syntax error")
	}

	if !intervalRegExp.MatchString(interval) {
		return false, errors.New("chronos: schedule: interval field syntax error")
	}

	return true, nil
}

func (j *Job) AddEnvVar(name, value string) *Job {
	j.EnvironmentVariables = append(j.EnvironmentVariables, &EnvVar{
		Name:  name,
		Value: value,
	})
	return j
}

func (j *Job) AddUri(uri string) *Job {
	j.Uris = append(j.Uris, uri)
	return j
}

func (j *Job) Type() int {
	hasParents := j.Parents != nil && len(j.Parents) != 0
	hasSchedule := j.Schedule != ""

	if hasParents && !hasSchedule {
		return JOB_TYPE_DependencyBased
	} else if !hasParents && hasSchedule {
		return JOB_TYPE_ScheduleBased
	} else {
		return JOB_TYPE_Unknown
	}
}

func (c *Client) Jobs() ([]*Job, error) {
	var jobs []*Job

	if err := c.apiGet("/scheduler/jobs", nil, &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (c *Client) Job(name string) (*Job, error) {
	jobs, err := c.Jobs()
	if err != nil {
		return nil, err
	}

	for _, j := range jobs {
		if j.Name == name {
			return j, nil
		}
	}

	return nil, errors.New("job does not exist")
}

func (c *Client) RunJob(name string) error {
	return c.apiPut(fmt.Sprintf("/scheduler/job/%s", name), nil, nil)
}

func (c *Client) DeleteJob(name string) error {
	return c.apiDelete(fmt.Sprintf("/scheduler/job/%s", name), nil, nil)
}

func (c *Client) KillJob(name string) error {
	return c.apiDelete(fmt.Sprintf("/scheduler/task/kill/%s", name), nil, nil)
}

func (c *Client) CreateJob(job *Job) error {
	var url string

	switch job.Type() {
	case JOB_TYPE_DependencyBased:
		url = "/scheduler/dependency"
	case JOB_TYPE_ScheduleBased:
		url = "/scheduler/iso8601"
	case JOB_TYPE_Unknown:
		return errors.New("job must include one of parents and schedule")
	default:
		panic("unreachable")
	}

	return c.apiPost(url, job, nil)
}

func (c *Client) UpdateJob(job *Job) error {
	var url string

	switch job.Type() {
	case JOB_TYPE_DependencyBased:
		url = "/scheduler/dependency"
	case JOB_TYPE_ScheduleBased:
		url = "/scheduler/iso8601"
	case JOB_TYPE_Unknown:
		return errors.New("job must include one of parents and schedule")
	default:
		panic("unreachable")
	}

	return c.apiPut(url, job, nil)
}
