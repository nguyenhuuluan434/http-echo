package main

import (
	"errors"
	"fmt"
	"github.com/myteksi/hystrix-go/hystrix"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	url := "http://localhost:8080/"
	hystrix.ConfigureCommand("callService", hystrix.CommandConfig{
		Timeout:                     1000,
		MaxConcurrentRequests:       100,
		ErrorPercentThreshold:       25,
		QueueSizeRejectionThreshold: 100,
	})
	for i := 0; i < 100; i++ {
		j := i
		hystrix.Do("callService", func() error {
			fmt.Println("Request number:", j)
			resp, _ := http.Get(url)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				fmt.Println(string(body))
				return errors.New("error from server")
			}
			defer resp.Body.Close()
			fmt.Println(string(body))
			return nil
		}, func(e error) error {
			log.Println(e)
			return e
		})
	}
}
