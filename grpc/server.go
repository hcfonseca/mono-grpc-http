package main

import (
	"crypto/tls"
	"crypto/x509"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/hcfonseca/testing-grpc/pb"
)

type server struct {
	pb.UnimplementedPayloadServiceServer
}

func main() {
	lis, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	caPem, err := ioutil.ReadFile("./cert/localhost.crt")
	if err != nil {
		log.Fatal(err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPem) {
		log.Fatal(err)
	}

	serverCert, err := tls.LoadX509KeyPair("./cert/localhost.crt", "./cert/localhost.decrypted.key")
	if err != nil {
		log.Fatal(err)
	}

	conf := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}
	tlsCredentials := credentials.NewTLS(conf)
	s := grpc.NewServer(grpc.Creds(tlsCredentials))

	pb.RegisterPayloadServiceServer(s, &server{})
	log.Println("Starting gRPC server")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *server) GetPayload(_ context.Context, p *pb.Payload) (*pb.Payload, error) {
	p.Message = "Ola Mundo"
	return p, nil
}
