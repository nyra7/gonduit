module server_main

go 1.26

replace server => ../../internal/server

replace shared => ../../internal/shared

require (
	server v0.0.0
	shared v0.0.0
)

require (
	github.com/creack/pty v1.1.24 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260311181403-84a4fc48630c // indirect
	google.golang.org/grpc v1.79.2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
