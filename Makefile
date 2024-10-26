build_proto:
	PATH=$PATH:~/go/bin
	protoc --go_out=./csi_proto --go_opt=paths=source_relative --go-grpc_out=./csi_proto --go-grpc_opt=paths=source_relative csi.proto
	protoc --go_out=./storageagent_proto --go_opt=paths=source_relative --go-grpc_out=./storageagent_proto --go-grpc_opt=paths=source_relative storage_agent.proto

build_storageagent:
	cd storageagent
	CGO_ENABLED=0 GOOS=linux go build -o storageagent
	strip storageagent
	cd ..

