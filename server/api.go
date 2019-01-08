package server

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	grpcRuntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"net"
	"net/http"
	"spaceship/apigrpc"
	"strings"
)

type Server struct {
	grpcServer           *grpc.Server
	grpcGatewayServer    *http.Server
}

func StartServer() *Server {

	port := 7350

	grpcServer := grpc.NewServer()

	s := &Server{
		grpcServer: grpcServer,
	}

	apigrpc.RegisterSpaceShipServer(grpcServer, s)

	log.Printf("Starting server for gRPC requests on port %d", port-1)
	go func(){
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port-1))
		if err != nil {
			log.Fatal("Error while creating listener for gRPC server")
		}
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal("Error while binding listener to gRPC server")
		}
	}()

	// Start Gateway server now
	ctx := context.Background()
	grpcGateway := grpcRuntime.NewServeMux(
		grpcRuntime.WithMetadata(func (ctx context.Context, r *http.Request) metadata.MD {
			// For RPC GET operations pass through any custom query parameters.
			if r.Method != "GET" || !strings.HasPrefix(r.URL.Path, "/v1/rpc/") {
				return metadata.MD{}
			}

			q := r.URL.Query()
			p := make(map[string][]string, len(q))
			for k, vs := range q {
				p["q_"+k] = vs
			}
			return metadata.MD(p)
		}),
	)

	dialAddr := fmt.Sprintf("127.0.0.1:%d", port-1)
	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	if err := apigrpc.RegisterSpaceShipHandlerFromEndpoint(ctx, grpcGateway, dialAddr, dialOpts); err != nil {
		log.Fatal("Error while registering gateway to gRPC server", err)
	}

	grpcGatewayRouter := mux.NewRouter()
	// Special case routes. Do NOT enable compression on WebSocket route, it results in "http: response.Write on hijacked connection" errors.
	grpcGatewayRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }).Methods("GET")
	//grpcGatewayRouter.HandleFunc("/ws", NewSocketWsAcceptor(logger, config, sessionRegistry, matchmaker, tracker, jsonpbMarshaler, jsonpbUnmarshaler, pipeline)).Methods("GET")


	grpcGatewayRouter.NewRoute().HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do not allow clients to set certain headers.
		// Currently disallowed headers:
		// "Grpc-Timeout"
		r.Header.Del("Grpc-Timeout")

		// Allow GRPC Gateway to handle the request.
		grpcGateway.ServeHTTP(w, r)
	})

	// Enable CORS on all requests.
	CORSHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type", "User-Agent"})
	CORSOrigins := handlers.AllowedOrigins([]string{"*"})
	CORSMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE"})
	handlerWithCORS := handlers.CORS(CORSHeaders, CORSOrigins, CORSMethods)(grpcGatewayRouter)

	// Set and start gRPC Gateway
	s.grpcGatewayServer = &http.Server{
		MaxHeaderBytes: 5120,
		Handler:        handlerWithCORS,
	}

	log.Printf("Starting gateway server for HTTP requests on port %d", port)
	go func(){
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatal("Error while creating listener for gateway server", err)
		}
		if err := s.grpcGatewayServer.Serve(listener); err != nil {
			log.Fatal("Error while serving gRPC gateway server", err)
		}
	}()

	return s

}

func decompressHandler( h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				break
			}
			r.Body = gr
		case "deflate":
			r.Body = flate.NewReader(r.Body)
		default:
			// No request compression.
		}
		h.ServeHTTP(w, r)
	})
}

