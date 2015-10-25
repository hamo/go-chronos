package chronos

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Cluster struct {
	sync.RWMutex

	Url string

	Protocol string

	Members []*Member
	Current *Member
}

type Member struct {
	Host   string
	Active bool
}

func newCluster(chronosUrl string) (*Cluster, error) {
	chronos, err := url.Parse(chronosUrl)
	if err != nil {
		return nil, err
	}
	if chronos.Scheme != "http" && chronos.Scheme != "https" {
		return nil, fmt.Errorf("chronos: cluster url scheme %s is not supported", chronos.Scheme)
	}

	re := new(Cluster)
	re.Url = chronosUrl
	re.Protocol = chronos.Scheme
	re.Members = make([]*Member, 0)
	for _, addr := range strings.Split(chronos.Host, ",") {
		re.Members = append(re.Members,
			&Member{
				Host:   addr,
				Active: true,
			})
	}
	re.Current = re.Members[0]

	return re, nil
}

func (c *Cluster) GetMember() (string, error) {
	c.RLock()
	defer c.RUnlock()

	if c.Current == nil || !c.Current.Active {
		return "", errors.New("chronos: no available cluster member")
	}
	return c.GenerateChronosUrl(c.Current), nil
}

func (c *Cluster) GenerateChronosUrl(member *Member) string {
	return fmt.Sprintf("%s://%s", c.Protocol, member.Host)
}

func (c *Cluster) MarkInactive() {
	c.Lock()
	defer c.Unlock()

	member := c.Current
	member.Active = false
	c.Current = nil

	go func() {
		for {
			resp, err := http.Get(c.GenerateChronosUrl(member) + "/ping")
			if err == nil && resp.StatusCode == 200 {
				member.Active = true
				return
			}

			<-time.After(time.Duration(5 * time.Second))
		}
	}()

	for _, m := range c.Members {
		if m.Active {
			c.Current = m
		}
	}
}
