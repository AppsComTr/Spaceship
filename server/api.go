package server

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/globalsign/mgo"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	grpcRuntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"net/http"
	"spaceship/apigrpc"
	"strings"
)

type ctxUserIDKey struct{}
type ctxUsernameKey struct{}
type ctxExpiryKey struct{}
type ctxFullMethodKey struct{}

type Server struct {
	db 					 *mgo.Session
	grpcServer           *grpc.Server
	grpcGatewayServer    *http.Server
	config *Config
	gameHolder *GameHolder
	leaderboard *Leaderboard
}

func (s *Server) Stop() {
	if err := s.grpcGatewayServer.Shutdown(context.Background()); err != nil {
		log.Println("Couldn't shutdown grpc gateway server")
	}

	s.grpcServer.GracefulStop()
}

func StartServer(sessionHolder *SessionHolder, gameHolder *GameHolder, config *Config, jsonProtoMarshler *jsonpb.Marshaler, jsonProtoUnmarshler *jsonpb.Unmarshaler, pipeline *Pipeline, db *mgo.Session, leaderboard *Leaderboard) *Server {

	port := config.Port

	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			nCtx, err := securityInterceptorFunc(ctx, req, info, config)
			if err != nil {
				return nil, err
			}
			ctx = nCtx
			return handler(ctx, req)
		}),
	}

	grpcServer := grpc.NewServer(serverOpts...)

	s := &Server{
		grpcServer: grpcServer,
		config: config,
		gameHolder: gameHolder,
		db: db,
		leaderboard: leaderboard,
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
		grpcRuntime.WithMarshalerOption(grpcRuntime.MIMEWildcard, &grpcRuntime.JSONPb{OrigName: true, EmitDefaults: true}),
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
	grpcGatewayRouter.HandleFunc("/ws", NewSocketAcceptor(sessionHolder, config, gameHolder, jsonProtoMarshler, jsonProtoUnmarshler, pipeline)).Methods("GET")
	fs := http.FileServer(http.Dir("/var/spaceshipassets/"))
	grpcGatewayRouter.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fs))

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
		if err := s.grpcGatewayServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatal("Error while serving gRPC gateway server", err)
		}
	}()

	return s

}

func securityInterceptorFunc(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, config *Config) (context.Context, error) {
	switch info.FullMethod {
	case "/spaceship.api.SpaceShip/AuthenticateFacebook":
		fallthrough
	case "/spaceship.api.SpaceShip/AuthenticateFingerprint":
		//No security everyone can make request to this endpoint
		return ctx, nil
	default:
		// This endpoint requires autrhozation with bearer
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Println("Cannot extract metadata from incoming context")
			return nil, status.Error(codes.FailedPrecondition, "Cannot extract metadata from incoming context")
		}
		auth, ok := md["authorization"]
		if !ok {
			auth, ok = md["grpcgateway-authorization"]
		}
		if !ok {
			// Neither "authorization" nor "grpc-authorization" were supplied.
			return nil, status.Error(codes.Unauthenticated, "Auth token required")
		}
		if len(auth) != 1 {
			// Value of "authorization" or "grpc-authorization" was empty or repeated.
			return nil, status.Error(codes.Unauthenticated, "Auth token invalid")
		}
		userID, username, exp, ok := parseBearerAuth([]byte(config.AuthConfig.JWTSecret), auth[0])
		if !ok {
			// Value of "authorization" or "grpc-authorization" was malformed or expired.
			return nil, status.Error(codes.Unauthenticated, "Auth token invalid")
		}
		ctx = context.WithValue(context.WithValue(context.WithValue(ctx, ctxUserIDKey{}, userID), ctxUsernameKey{}, username), ctxExpiryKey{}, exp)
	}
	return context.WithValue(ctx, ctxFullMethodKey{}, info.FullMethod), nil
}

func parseBearerAuth(hmacSecretByte []byte, auth string) (userID string, username string, exp int64, ok bool) {
	if auth == "" {
		return
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	return parseToken(hmacSecretByte, string(auth[len(prefix):]))
}

func parseToken(hmacSecretByte []byte, tokenString string) (userID string, username string, exp int64, ok bool) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if s, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || s.Hash != crypto.SHA256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return hmacSecretByte, nil
	})
	if err != nil {
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return
	}
	userID, ok = claims["uid"].(string)
	if !ok {
		return
	}
	return userID, claims["usn"].(string), int64(claims["exp"].(float64)), true
}

func ConnectDB(config *Config) *mgo.Session {

	conn, err := mgo.Dial(config.DBConfig.ConnString)
	if err != nil {
		log.Fatal("Cannot dial mongo", err)
	}
	log.Println("Mongo connection completed")
	return conn

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

