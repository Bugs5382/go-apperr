module github.com/Bugs5382/go-apperr/apperrgrpc

go 1.26.0

require (
	github.com/Bugs5382/go-apperr v1.2.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260928230214-8a89bd6388cc
	google.golang.org/grpc v1.84.0
	google.golang.org/protobuf v1.36.12
)

require golang.org/x/sys v0.47.0 // indirect

replace github.com/Bugs5382/go-apperr => ../
