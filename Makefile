.PHONY: gen gen-api gen-rpc up

gen: gen-api gen-rpc

gen-api:
	goctl api go -api desc/gateway.api -dir app/gateway

gen-rpc:
	for svc in user rate ranking article interaction favorite; do \
		goctl rpc protoc desc/$$svc.proto --go_out=. --go-grpc_out=. --zrpc_out=app/$$svc/cmd/rpc; \
	done

up:
	docker compose up -d