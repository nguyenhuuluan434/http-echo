package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"grabvn-golang-bootcamp/week_6/http-echo/server/config"
	"math/rand"
	"net"
	"net/http"
)

const (
	RequestIdKey     string = "X-Request-Id"
	StatusCode       string = "status_code"
	RequestUri       string = "request_uri"
	EtcServicePrefix string = "service-i-am-learning"
	InstanceIP       string = "instance_ip"
	InstanceID       string = "instance_id"
)

type bindAddress struct {
	host string
	port int
}

func (b *bindAddress) GetHost() string {
	return b.host
}

func (b *bindAddress) GetPort() int {
	return b.port
}

type Address interface {
	GetHost() string
	GetPort() int
}

func CreateAddress(host string, port int) Address {
	return &bindAddress{host: host, port: port}
}

type server struct {
	logger               *log.Logger
	requestInfoGenerator RequestInfoGenertor
	handler              http.Handler
	logInfo              LogInfo
	host                 string
	port                 int
	config               *config.Configuration
	dataDogClient        *statsd.Client
	etcdClient           EtcdClient
	instanceId           string
	address              string
}

func (s *server) log() {
	fields := getFields(s.logInfo)
	s.logger.WithFields(fields).Log(s.logger.Level)
	return
}

func getFields(obj interface{}) (fields log.Fields) {
	b, _ := json.Marshal(obj)
	json.Unmarshal(b, &fields)
	return
}

func CreateServer(address Address, config *config.Configuration, logger *log.Logger, requestInfoGenerator RequestInfoGenertor, logInfo LogInfo, dataDogClient *statsd.Client) *server {
	return &server{config: config, logger: logger, requestInfoGenerator: requestInfoGenerator, logInfo: logInfo, host: address.GetHost(), port: address.GetPort(), dataDogClient: dataDogClient, instanceId: uuid.New().String()}
}

type Option func(server *server)

func WithAddress(address Address) Option {
	return func(server *server) {
		server.port = address.GetPort()
		server.host = address.GetHost()
	}
}

func WithConfiguration(config *config.Configuration) Option {
	return func(server *server) {
		server.config = config
	}
}

func WithLogger(logger *log.Logger) Option {
	return func(server *server) {
		server.logger = logger
	}
}

func WithRequestGenerator(requestInfoGenerator RequestInfoGenertor) Option {
	return func(server *server) {
		server.requestInfoGenerator = requestInfoGenerator
	}
}

func WithLogInfo(logInfo LogInfo) Option {
	return func(server *server) {
		server.logInfo = logInfo
	}
}
func WithDataDogClient(dataDogClient *statsd.Client) Option {
	return func(server *server) {
		server.dataDogClient = dataDogClient
	}
}

func NewServer(options ...Option) (*server, error) {
	server := &server{}
	for _, o := range options {
		o(server)
	}
	//check require option
	server.instanceId = uuid.New().String()
	return server, nil
}

func (s *server) SetEtcClient(client EtcdClient) *server {
	s.etcdClient = client
	return s
}

func (s *server) flushLogLine(obj interface{}) {
	message, _ := json.Marshal(obj)
	s.logger.Log(s.logger.Level, string(message))
}

func (s *server) ListenAndServe() {
	s.flushLogLine(fmt.Sprintf("echo server is starting on port %s:%d...", s.host, s.port))
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	s.address = listener.Addr().String()
	server := &http.Server{}
	router := http.NewServeMux()
	router.HandleFunc("/", s.echo)
	server.Handler = s.repairHandleRequest(router)
	s.logger.Log(s.logger.Level, listener.Addr().String())
	//add to etcd
	go s.etcdClient.RegisService(EtcServicePrefix, listener.Addr().String())
	server.Serve(listener)
}

// Echo echos back the request as a response
func (s *server) echo(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Access-Control-Allow-Headers", "Content-Range, Content-Disposition, Content-Type, ETag")
	//s.logger.Log(s.logger.Level,request.Header.Get(RequestIdKey))
	// 30% chance of failure
	var statusCode int
	if rand.Intn(100) < 30 {
		statusCode = http.StatusInternalServerError
		writer.WriteHeader(statusCode)
		writer.Write([]byte("a chaos monkey broke your server"))
		s.dataDogClient.Count("internalServerError", 1, []string{""}, 1)
	} else {
		// Happy path
		statusCode = http.StatusOK
		writer.WriteHeader(statusCode)
		s.dataDogClient.Count("ok", 1, []string{""}, 1)
	}
	s.logInfo.AddProp(StatusCode, statusCode)
	request.Write(writer)
}

func (s *server) preHandleRequest(w http.ResponseWriter, r *http.Request) {
	requestId := r.Header.Get(RequestIdKey)
	if len(requestId) == 0 {
		requestId = s.requestInfoGenerator.GenerateRequestId().GetRequestId()
	}
	r.Header.Add(RequestIdKey, requestId)
	r.WithContext(context.WithValue(r.Context(), RequestIdKey, requestId))
	s.logInfo = s.logInfo.Init(r.Header.Get(RequestIdKey))
	s.logInfo.AddProp(RequestUri, r.RequestURI)
	s.logInfo.AddProp(InstanceIP, s.address)
	s.logInfo.AddProp(InstanceID, s.instanceId)
	return
}

func (s *server) postHandleRequest(w http.ResponseWriter, r *http.Request) {
	requestId := r.Header.Get(RequestIdKey)
	if len(requestId) > 0 {
		r.Header.Del(RequestIdKey)
	}
	s.logInfo.Elapsed()
	s.log()
	return

}

func (s *server) repairHandleRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			s.preHandleRequest(w, r)
			defer s.postHandleRequest(w, r)
			h.ServeHTTP(w, r)
		})
}
