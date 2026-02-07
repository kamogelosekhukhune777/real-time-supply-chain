# Define dependencies

BASE_MODULE     := github.com/kamogelosekhukhune777/real-time-supply-chain

# ============================================================================================================================================
#Code gen(s)

generate: generate-inventory-v1

generate-inventory-v1:
	protoc \
	  --proto_path=. \
	  --go_out=services/inventory/internal/service/events \
	  --go_opt=module=github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events \
	  --go_opt=Mcontracts/common/v1/metadata.proto=github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/common/v1 \
	  --go_opt=Mcontracts/inventory/v1/inventory_events.proto=github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/inventory/v1 \
	  --go_opt=Mcontracts/sales/v1/sale_events.proto=github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/sales/v1 \
	  --go_opt=Mcontracts/shipment/v1/shipment_events.proto=github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/shipment/v1 \
	  --go_opt=Mcontracts/optimization/v1/optimization_events.proto=github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/optimization/v1 \
	  --go_opt=Mcontracts/demand/v1/forecast_events.proto=github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/demand/v1 \
	  --go_opt=Mcontracts/order/v1/order_events.proto=github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/order/v1 \
	  contracts/common/v1/metadata.proto \
	  contracts/inventory/v1/inventory_events.proto \
	  contracts/sales/v1/sale_events.proto \
	  contracts/shipment/v1/shipment_events.proto \
	  contracts/optimization/v1/optimization_events.proto \
	  contracts/demand/v1/forecast_events.proto \
	  contracts/order/v1/order_events.proto

# ============================================================================================================================================
# Docker Compose

compose-up:
	cd ./zarf/compose/ && docker compose -f docker_compose.yaml -p compose up -d

compose-build-up: build compose-up

compose-down:
	cd ./zarf/compose/ && docker compose -f docker_compose.yaml down

compose-down-v:
	cd ./zarf/compose/ && docker compose -f docker_compose.yaml down -v

compose-logs:
	cd ./zarf/compose/ && docker compose -f docker_compose.yaml logs

compose-logs-v:
	cd ./zarf/compose/ && docker compose -f docker_compose.yaml logs -v

compose-ps:
	cd ./zarf/compose/ && docker compose -f docker_compose.yaml ps

compose-ps-v:
	cd ./zarf/compose/ && docker compose -f docker_compose.yaml ps -v

# ============================================================================================================================================
# Hitting endpoints



# ============================================================================================================================================
# Modules support

deps-reset:
	git checkout -- go.mod
	go mod tidy
	go mod vendor

tidy:
	go mod tidy
	go mod vendor

deps-list:
	go list -m -u -mod=readonly all

deps-upgrade:
	go get -u -v ./...
	go mod tidy
	go mod vendor

deps-cleancache:
	go clean -modcache

list:
	go list -mod=mod all

# ============================================================================================================================================
#Run

shipment:
	go run services/shipment/cmd/shipment/main.go

inventory:
	go run services/inventory/cmd/inventorymain.go
