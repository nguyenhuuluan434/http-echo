package main

import (
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	url := "http://localhost:8080/"
	var body []byte
	var err error
	for i := 0; i < 100; i++ {
		j := i
		retry.Do(
			func() error {
				fmt.Println("Request number:", j)
				resp, _ := http.Get(url)
				body, err = ioutil.ReadAll(resp.Body)
				if resp.StatusCode != 200 {
					return errors.New("error was occurred at destination server")
				}
				defer resp.Body.Close()
				if err != nil {
					return err
				}
				return nil
			},
			retry.OnRetry(func(n uint, err error) {
				log.Printf("#%d: %s\n", n, err)
			}),
			retry.DelayType(retry.FixedDelay),
			retry.Attempts(2),
		)
	}
}
