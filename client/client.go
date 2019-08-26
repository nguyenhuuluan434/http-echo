package main

import (
	"errors"
	"fmt"
	"github.com/myteksi/hystrix-go/hystrix"
	"http-echo/server"
	"http-echo/server/config"
	"io/ioutil"
	"log"
	"net/http"

	//"io/ioutil"
	//"net/http"
	"sync"
	"time"
)

var (
	etcdClient       server.EtcdClient
	serviceInstances ServiceInstances
)

func init() {
	configPath := "./service.yml"
	config := config.Loadconfig(configPath)
	etcdClient = server.NewEtcdClient(config.Etcd.Endpoints, config.Etcd.Timeout)
	serviceInstances = ServiceInstances{instances: map[string]ServiceInstance{}, endpoints: make([]string, 0), next: 0}
	WatchServiceInstances(server.EtcServicePrefix, etcdClient, &serviceInstances)
	serviceInstances.RemoveServiceInstanceBackground()
}

type ServiceInstance struct {
	location         string
	enable           bool
	timeChangeStatus time.Time
}

type ServiceInstances struct {
	instances map[string]ServiceInstance
	mu        sync.Mutex
	endpoints []string
	next      int
}

func main() {
	time.Sleep(time.Second * 5)
	//setting default for hystrix
	setting := hystrix.Settings{
		Timeout:                     10000,
		MaxConcurrentRequests:       100,
		ErrorPercentThreshold:       15,
		QueueSizeRejectionThreshold: 100}
	hystrix.Initialize(&setting)
	circuitNamePrefix := "callEchoServer"

	for {
		url, err := serviceInstances.GetServiceInstance()

		if err != nil {
			log.Println(err)
			continue
		}
		circuitInstance := circuitNamePrefix + url
		hystrix.Do(circuitInstance, func() error {
			fmt.Print(" call to ", url)
			time.Sleep(10 * time.Millisecond)
			if err != nil {
				return err
			}
			resp, err := http.Get(url)
			if err != nil {
				return errors.New(" error call to destination ")
			}
			if resp.StatusCode > 200 {
				return errors.New(" destination service error ")
			}
			body, err := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			fmt.Println(" data return ", string(body))
			return nil
		}, func(err error) error {
			circuit, _, errGetCircuit := hystrix.GetCircuit(circuitInstance)
			if errGetCircuit != nil {
				return errGetCircuit
			}
			if circuit.IsOpen() {
				serviceInstances.DisableServiceInstance(url)
			}
			fmt.Println(err)
			return nil
		})
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(time.Hour)
}

func WatchServiceInstances(prefix string, etcdClient server.EtcdClient, serviceInstances *ServiceInstances) {
	go func(prefix string, etcdClient server.EtcdClient, serviceInstances *ServiceInstances) {
		for {
			locations, _ := etcdClient.GetServiceLocation(prefix)
			serviceInstances.mu.Lock()
			instances := serviceInstances.instances
			endpoints := make([]string, 0)
			endpoints = append(serviceInstances.endpoints)
			now := time.Now()
			for _, location := range locations {
				if _, ok := serviceInstances.instances[location]; !ok {
					instances[location] = ServiceInstance{location: location, enable: true, timeChangeStatus: now}
					endpoints = append(endpoints, location)
					continue
				}
				instance := instances[location]
				if instance.enable == false && getDiffTime(now, instance.timeChangeStatus, time.Minute) > int64(5) {
					instance.enable = true
					instance.timeChangeStatus = now
					serviceInstances.endpoints = append(serviceInstances.endpoints, location)
				}
				instances[location] = instance
			}
			serviceInstances.instances = instances
			serviceInstances.endpoints = endpoints
			serviceInstances.mu.Unlock()
			time.Sleep(10 * time.Second)
		}
	}(prefix, etcdClient, serviceInstances)

	return
}

func getDiffTime(future, past time.Time, duration time.Duration) int64 {
	switch duration {
	case time.Second:
		return int64(future.Sub(past).Seconds())
	case time.Minute:
		return int64(future.Sub(past).Minutes())
	default:
		return int64(future.Sub(past).Hours())
	}
}

func (s *ServiceInstances) DisableServiceInstance(location string) {
	fmt.Println("disable service instance ", location)
	s.mu.Lock()
	if v, ok := s.instances[location]; ok {
		v.enable = false
		s.instances[location] = v
	}
	s.mu.Unlock()
}

func (s *ServiceInstances) RemoveServiceInstanceWithLocation(location string) {
	fmt.Println("remove service instance", location)
	s.mu.Lock()
	if _, ok := s.instances[location]; ok {
		delete(s.instances, location)
		s.endpoints = removeItemFromList(s.endpoints, location)
	}
	s.mu.Unlock()
}

func (s *ServiceInstances) RemoveServiceInstanceBackground() {
	go func(s *ServiceInstances) {
		for {
			if len(s.instances) > 0 {
				now := time.Now()
				endpoints := s.endpoints
				s.mu.Lock()
				for k, v := range s.instances {
					if v.enable == true {
						continue
					}
					if getDiffTime(now, v.timeChangeStatus, time.Second) >= 10 {
						delete(s.instances, k)
						s.endpoints = removeItemFromList(endpoints, k)
					}
				}
				s.mu.Unlock()
			}
			time.Sleep(5 * time.Second)
		}
	}(s)
}
func (s *ServiceInstances) GetServiceInstance() (location string, err error) {
	fmt.Println("get service instance")
	retry := 0
	for {
		if retry > 3 {
			break
		}
		s.mu.Lock()
		if len(s.instances) <= 0 || len(s.endpoints) <= 0 {
			s.mu.Unlock()
			retry++
			continue
		}
		if s.next > len(s.endpoints) {
			s.next = len(s.endpoints) - 1
		}
		sc := s.endpoints[s.next]
		s.next = (s.next + 1) % len(s.endpoints)
		s.mu.Unlock()
		if s.instances[sc].enable == false {
			retry++
			continue
		}
		return sc, nil
	}
	return location, errors.New("could not find any instance is running")

}

func removeItemFromList(input []string, item string) (output []string) {
	for i := 0; i < len(input); i++ {
		if input[i] == item {
			input = append(input[:i], input[i+1:]...)
			i--
		}
	}
	return input
}
