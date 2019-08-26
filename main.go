package main

import (
	"flag"
	"github.com/DataDog/datadog-go/statsd"
	log "github.com/sirupsen/logrus"
	"grabvn-golang-bootcamp/week_6/http-echo/server"
	"grabvn-golang-bootcamp/week_6/http-echo/server/config"
	"os"
)

var (
	hostParam     string
	portParam     int
	logLevelParam string
	configPath    string
	logFile       string
	client        *statsd.Client
	err           error
)

func init() {
	client, err = statsd.New("127.0.0.1:8125",
		statsd.WithNamespace("localhost"),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.StringVar(&hostParam, "host", "127.0.0.1", "the ip will listen")
	flag.IntVar(&portParam, "port", 0, "the port will bind")
	flag.StringVar(&logLevelParam, "logLevel", "Info", "the log level")
	flag.StringVar(&configPath, "configPath", "./service.yml", "the config file location")
	flag.StringVar(&logFile, "logFile", "./service.log", "the log file location")
	flag.Parse()
	config := config.Loadconfig(configPath)
	// Create our servers
	logger := log.New()
	logger.SetFormatter(&log.JSONFormatter{})
	logLevel, err := log.ParseLevel(logLevelParam)
	if err != nil {
		logLevel = log.InfoLevel
	}
	logger.Level = logLevel
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	logger.SetOutput(f)

	requestInfoGenerator := server.CreateRequestInfoGenertor()
	logInfo := server.CreateLogInfo()

	etcdClient := server.NewEtcdClient(config.Etcd.Endpoints, config.Etcd.Timeout)

	bindAddress := server.CreateAddress(hostParam, portParam)
	s := server.CreateServer(bindAddress, config, logger, requestInfoGenerator, logInfo, client)
	s.SetEtcClient(etcdClient)
	// Start the server
	s.ListenAndServe()
}
