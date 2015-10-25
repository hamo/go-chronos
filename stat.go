package chronos

import (
	"fmt"
)

type JobStatHistogram struct {
	Percentile75th float32 `json:"75thPercentile"`
	Percentile95th float32 `json:"95thPercentile"`
	Percentile98th float32 `json:"98thPercentile"`
	Percentile99th float32 `json:"99thPercentile"`
	Median         float32 `json:"Median"`
	Mean           float32 `json:"mean"`

	Count int `json:"count"`
}

type JobStatTasksHistory struct {
	TaskId  string `json:"taskId"`
	JobName string `json:"jobName"`
	SlaveId string `json:"slaveId"`

	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Duration  string `json:"duration"`

	Status string `json:"status"`

	NumElementsProcessed int `json:"numElementsProcessed"`
}

type JobStat struct {
	Histogram *JobStatHistogram `json:"histogram"`

	TaskStatHistory []*JobStatTasksHistory `json:"taskStatHistory,omitempty"`
}

func (c *Client) GetJobStat(job string) (*JobStat, error) {
	js := new(JobStat)

	if err := c.apiGet(fmt.Sprintf("/scheduler/job/stat/%s", job), nil, &js); err != nil {
		return nil, err
	}

	return js, nil
}

//  /scheduler/stats/99thPercentile
//  /scheduler/stats/98thPercentile
//  /scheduler/stats/95thPercentile
//  /scheduler/stats/75thPercentile
//  /scheduler/stats/median
//  /scheduler/stats/mean
