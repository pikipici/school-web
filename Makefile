SHELL := /bin/sh

.PHONY: test-backend vet-backend build-backend build-adminctl lint-frontend typecheck-frontend test-frontend build-frontend verify deploy

test-backend:
	cd backend && go test ./... -count=1

vet-backend:
	cd backend && go vet ./...

build-backend:
	cd backend && mkdir -p bin && go build -o bin/api ./cmd/api

build-adminctl:
	cd backend && mkdir -p bin && go build -o bin/adminctl ./cmd/adminctl

lint-frontend:
	cd frontend && npm run lint

typecheck-frontend:
	cd frontend && npm run typecheck

test-frontend:
	cd frontend && npm test

build-frontend:
	cd frontend && npm run build

verify: test-backend vet-backend build-backend build-adminctl lint-frontend typecheck-frontend test-frontend build-frontend

deploy:
	bash deploy/deploy.sh
