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
	"github.com/mediocregopher/radix/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
	"net/http"
	"spaceship/apigrpc"
	"strings"
	"time"
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
	stats *Stats
	logger *Logger
}

func (s *Server) Stop() {
	if err := s.grpcGatewayServer.Shutdown(context.Background()); err != nil {
		s.logger.Errorw("Couldn't shutdown grpc gateway server", "error", err)
	}

	s.grpcServer.GracefulStop()
}

func StartServer(sessionHolder *SessionHolder, gameHolder *GameHolder, config *Config, jsonProtoMarshler *jsonpb.Marshaler, jsonProtoUnmarshler *jsonpb.Unmarshaler, pipeline *Pipeline, db *mgo.Session, leaderboard *Leaderboard, stats *Stats, logger *Logger) *Server {

	port := config.Port

	//We should set middleware to check token in the header is valid for endpoints which required authorization
	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			stats.IncrRequest()
			//Requests will be passed to our interceptor function to verify
			nCtx, err := securityInterceptorFunc(ctx, req, info, config, logger)
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
		stats: stats,
		logger: logger,
	}

	//Grpc server is registered to our implementation of server struct
	apigrpc.RegisterSpaceShipServer(grpcServer, s)

	//Grpc server can be started, but it should be work in another routine not to block current one
	//We'll define grpc gateway server also
	logger.Infof("Starting server for gRPC requests on port %d", port-1)
	go func(){
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port-1))
		if err != nil {
			logger.Fatal("Error while creating listener for gRPC server")
		}
		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatal("Error while binding listener to gRPC server")
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
		//This marshaler option is used to prevent not serializing of fields with default values
		grpcRuntime.WithMarshalerOption(grpcRuntime.MIMEWildcard, &grpcRuntime.JSONPb{OrigName: true, EmitDefaults: true}),
	)

	dialAddr := fmt.Sprintf("127.0.0.1:%d", port-1)
	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	if err := apigrpc.RegisterSpaceShipHandlerFromEndpoint(ctx, grpcGateway, dialAddr, dialOpts); err != nil {
		logger.Fatalw("Error while registering gateway to gRPC server", "error", err)
	}

	//Router should be created to define custom endpoints. One of them will be used for web socket connections.
	grpcGatewayRouter := mux.NewRouter()
	grpcGatewayRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }).Methods("GET")
	grpcGatewayRouter.HandleFunc("/ws", NewSocketAcceptor(sessionHolder, config, gameHolder, jsonProtoMarshler, jsonProtoUnmarshler, pipeline, stats, logger)).Methods("GET")
	grpcGatewayRouter.Handle("/metrics", stats.prometheusExporter)
	//Server should also serve static files (like avatar etc..), so need to define file server on router
	fs := http.FileServer(http.Dir("/var/spaceshipassets/"))
	grpcGatewayRouter.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fs))

	//First request body need to be decompressed if data was compressed with gzip
	//And compress response body if client supports it
	//And check limit max body size in requests
	handlerDecompress := decompressHandler(grpcGateway)
	handlerCompressResponse := handlers.CompressHandler(handlerDecompress)
	handlerMaxBodySize := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxRequestBodySize)
		handlerCompressResponse.ServeHTTP(w, r)
	})

	//All other requests should be handled by grpc gateway server.
	grpcGatewayRouter.NewRoute().HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//We can delete or modify headers before passing request to gateway server.
		r.Header.Del("Grpc-Timeout")

		// Pass request to gateway server
		handlerMaxBodySize.ServeHTTP(w, r)
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

	//Gateway server also should be started in another routine not to block current one
	logger.Infof("Starting gateway server for HTTP requests on port %d", port)
	go func(){
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			logger.Fatalw("Error while creating listener for gateway server", "error", err)
		}
		if err := s.grpcGatewayServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Fatalw("Error while serving gRPC gateway server", "error", err)
		}
	}()

	//TODO: we can subscribe to rabbitMQ queue here. We'll listen the queue, and according to retrieved data we'll check if user ids are in our session holder
	//TODO: if we'll find sessions we'll send data to this sessions
	//TODO: we can also make a module for this purpose with sending session holder module to it.
	//TODO: we also need to change session.Send method at everywhere that we used with new module's send method.

	return s

}

func securityInterceptorFunc(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, config *Config, logger *Logger) (context.Context, error) {
	switch info.FullMethod {
	case "/spaceship.api.SpaceShip/AuthenticateFacebook":
		fallthrough
	case "/spaceship.api.SpaceShip/AuthenticateFingerprint":
		//No security everyone can make request to this endpoint
		return ctx, nil
	//Requests that will be defined at the above won't be checked for authorization
	default:
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Errorw("Cannot extract metadata from incoming context", "context", ctx)
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

func ConnectDB(config *Config, logger *Logger) *mgo.Session {

	conn, err := mgo.DialWithTimeout(config.DBConfig.ConnString, time.Duration(30 * time.Second))
	if err != nil {
		logger.Fatalw("Cannot dial mongo", "error", err)
	}
	logger.Info("Mongo connection completed")
	return conn

}

func ConnectRedis(config *Config, logger *Logger) radix.Client{

	var redisClient radix.Client
	var err error

	if config.RedisConfig.CluesterEnabled {
		redisClient, err = radix.NewCluster([]string{config.RedisConfig.ConnString})
		if err != nil {
			logger.Fatalw("Redis Connection Failed", "error", err)
		}
	}else{
		redisClient, err = radix.NewPool("tcp", config.RedisConfig.ConnString, 1)
		if err != nil {
			logger.Fatalw("Redis Connection Failed", "error", err)
		}
	}
	return redisClient
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

