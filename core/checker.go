package core

import (
	"fmt"
	"github.com/hunterlong/statup/types"
	"github.com/hunterlong/statup/utils"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

type FailureData types.FailureData

func CheckServices() {
	CoreApp.Services, _ = SelectAllServices()
	utils.Log(1, fmt.Sprintf("Loaded %v Services", len(CoreApp.Services)))
	for _, v := range CoreApp.Services {
		obj := v
		go obj.StartCheckins()
		go obj.CheckQueue()
	}
}

func (s *Service) CheckQueue() {
	defer s.CheckQueue()
	s.Check()
	if s.Interval < 1 {
		s.Interval = 1
	}
	msg := fmt.Sprintf("Service: %v | Online: %v | Latency: %0.0fms", s.Name, s.Online, (s.Latency * 1000))
	utils.Log(1, msg)
	time.Sleep(time.Duration(s.Interval) * time.Second)
}

func (s *Service) Check() *Service {
	t1 := time.Now()
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	response, err := client.Get(s.Domain)
	response.Header.Set("User-Agent", "StatupMonitor")
	t2 := time.Now()
	s.Latency = t2.Sub(t1).Seconds()
	if err != nil {
		s.Failure(fmt.Sprintf("HTTP Error %v", err))
		return s
	}
	defer response.Body.Close()
	contents, _ := ioutil.ReadAll(response.Body)
	if s.Expected != "" {
		match, _ := regexp.MatchString(s.Expected, string(contents))
		if !match {
			s.LastResponse = string(contents)
			s.LastStatusCode = response.StatusCode
			s.Failure(fmt.Sprintf("HTTP Response Body did not match '%v'", s.Expected))
			return s
		}
	}
	if s.ExpectedStatus != response.StatusCode {
		s.LastResponse = string(contents)
		s.LastStatusCode = response.StatusCode
		s.Failure(fmt.Sprintf("HTTP Status Code %v did not match %v", response.StatusCode, s.ExpectedStatus))
		return s
	}
	s.LastResponse = string(contents)
	s.LastStatusCode = response.StatusCode
	s.Online = true
	s.Record(response)
	return s
}

type HitData struct {
	Latency float64
}

func (s *Service) Record(response *http.Response) {
	s.Online = true
	s.LastOnline = time.Now()
	data := HitData{
		Latency: s.Latency,
	}
	s.CreateHit(data)
	OnSuccess(s)
}

func (s *Service) Failure(issue string) {
	s.Online = false
	data := FailureData{
		Issue: issue,
	}
	utils.Log(2, fmt.Sprintf("Service %v Failing: %v", s.Name, issue))
	s.CreateFailure(data)
	//SendFailureEmail(s)
	OnFailure(s)
}