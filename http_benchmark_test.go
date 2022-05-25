package testing_grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/hcfonseca/testing-grpc/pb"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"testing"
)

var client http.Client

const stopRequestPath = "STOP"
const workers = 4

type Request struct {
	Path    string
	Payload *pb.Payload
}

func init() {
	client = http.Client{}
}

func BenchmarkHTTP11(b *testing.B) {
	client.Transport = &http.Transport{
		TLSClientConfig: initTLSConfig(),
	}

	queue := make(chan Request)
	defer startWorkers(&queue, workers, startWorker)()
	b.ResetTimer() // don't count worker initialization time
	for i := 0; i < b.N; i++ {
		queue <- Request{Path: "https://localhost:8080"}
	}
}

func BenchmarkHTTP2(b *testing.B) {
	client.Transport = &http2.Transport{
		TLSClientConfig: initTLSConfig(),
	}

	queue := make(chan Request)
	defer startWorkers(&queue, workers, startWorker)()
	b.ResetTimer() // don't count worker initialization time
	for i := 0; i < b.N; i++ {
		queue <- Request{Path: "https://localhost:8080"}
	}
}

func BenchmarkGRPC(b *testing.B) {
	caCert, err := ioutil.ReadFile("cert/localhost.crt")
	if err != nil {
		log.Fatal(caCert)
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(caCert); !ok {
		log.Fatal(err)
	}

	clientCert, err := tls.LoadX509KeyPair("cert/localhost.crt", "cert/localhost.decrypted.key")
	if err != nil {
		log.Fatal(err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
	}

	conn, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(credentials.NewTLS(config)))
	if err != nil {
		log.Fatalf("Dial failed: %v", err)
	}

	client := pb.NewPayloadServiceClient(conn)
	requestQueue := make(chan Request)

	defer startWorkers(&requestQueue, workers, getStartGRPCWorkerFunction(client))()
	b.ResetTimer() // don't count worker initialization time

	for i := 0; i < b.N; i++ {
		requestQueue <- Request{Path: "http://localhost:9090", Payload: &pb.Payload{}}
	}
}

func initTLSConfig() *tls.Config {
	ca, err := ioutil.ReadFile("cert/localhost.crt")
	if err != nil {
		log.Fatalf("error getting cert: %s", err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(ca)

	return &tls.Config{
		RootCAs: caPool,
	}
}

func startWorkers(requestQueue *chan Request, noWorkers int, startWorker func(*chan Request, *sync.WaitGroup)) func() {
	var wg sync.WaitGroup
	for i := 0; i < noWorkers; i++ {
		startWorker(requestQueue, &wg)
	}
	return func() {
		wg.Add(noWorkers)
		stopRequest := Request{Path: stopRequestPath}
		for i := 0; i < noWorkers; i++ {
			*requestQueue <- stopRequest
		}
		wg.Wait()
	}
}

func startWorker(requestQueue *chan Request, wg *sync.WaitGroup) {
	go func() {
		for {
			request := <-*requestQueue
			if request.Path == stopRequestPath {
				wg.Done()
				return
			}
			get(request.Path)
		}
	}()
}

func get(path string) error {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		log.Println("error creating request ", err)
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println("error executing request ", err)
		return err
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("error reading response body ", err)
		return err
	}

	var response *pb.Payload
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		log.Println("error unmarshalling response ", err)
		return err
	}

	return nil
}

func getStartGRPCWorkerFunction(client pb.PayloadServiceClient) func(*chan Request, *sync.WaitGroup) {
	return func(requestQueue *chan Request, wg *sync.WaitGroup) {
		go func() {
			for {
				request := <-*requestQueue
				if request.Path == stopRequestPath {
					wg.Done()
					return
				}
				_, err := client.GetPayload(context.TODO(), request.Payload)
				if err != nil {
					log.Fatal(err)
				}
			}
		}()
	}
}
