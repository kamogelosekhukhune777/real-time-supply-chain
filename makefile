



# ============================================================================================================================================
# Docker Compose

# ============================================================================================================================================
# Modules support

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
#Code gen(s)

some:
	protoc --proto_path=. \
	       --proto_path=contracts \
	       --go_out=services/inventory/internal/domain/gen \
	       --go_opt=paths=source_relative \
	       contracts/inventory/inventory_events.proto

some2:
	protoc --proto_path=. \
	       --proto_path=contracts \
	       --go_out=services/inventory/internal/domain/gen \
	       --go_opt=module=github.com/kamogelosekhukhune777/real-time-supply-chain \
	       contracts/inventory/inventory_events.proto

generate:
	protoc --proto_path=. \
	       --go_out=. \
	       --go_opt=module=github.com/kamogelosekhukhune777/real-time-supply-chain \
	       contracts/common/metadata.proto \
	       contracts/inventory/inventory_events.proto

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
