SHELL := /bin/sh

.PHONY: test-backend vet-backend build-backend lint-frontend typecheck-frontend build-frontend verify deploy

test-backend:
	cd backend && go test ./... -count=1

vet-backend:
	cd backend && go vet ./...

build-backend:
	cd backend && mkdir -p bin && go build -o bin/api ./cmd/api

lint-frontend:
	cd frontend && npm run lint

typecheck-frontend:
	cd frontend && npm run typecheck

build-frontend:
	cd frontend && npm run build

verify: test-backend vet-backend build-backend lint-frontend typecheck-frontend build-frontend

deploy:
	@test -x deploy/deploy.sh || { echo "deploy/deploy.sh belum tersedia; selesaikan Task 0.9 terlebih dahulu." >&2; exit 1; }
	./deploy/deploy.sh
