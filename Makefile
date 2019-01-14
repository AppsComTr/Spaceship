protogen:
	protoc -I/usr/local/include -I. -I$$GOPATH/src -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway --go_out=plugins=grpc:. ./api/api.proto
	protoc -I/usr/local/include -I. -I$$GOPATH/src -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway --grpc-gateway_out=logtostderr=true:. ./api/api.proto
	protoc -I/usr/local/include -I. -I$$GOPATH/src -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway --go_out=plugins=grpc:. ./apigrpc/apigrpc.proto
	protoc -I/usr/local/include -I. -I$$GOPATH/src -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway --grpc-gateway_out=logtostderr=true:. ./apigrpc/apigrpc.proto


protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --swagger_out=logtostderr=true:. \
  ./apigrpc/apigrpc.proto