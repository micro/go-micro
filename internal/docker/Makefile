K6_IMAGE  := grafana/k6:latest
BASE_URL  ?= http://host.docker.internal:8080
ADMIN_USER ?= admin
ADMIN_PASS ?= micro
VUS       ?= 10
ITERATIONS ?= 50

.PHONY: run test stress

run:
	docker-compose up -d --force-recreate --remove-orphans --build

test:
	docker run --rm \
		-e BASE_URL=$(BASE_URL) \
		-e MICRO_ADMIN_USER=$(ADMIN_USER) \
		-e MICRO_ADMIN_PASSWORD=$(ADMIN_PASS) \
		-e VUS=$(VUS) \
		-e ITERATIONS=$(ITERATIONS) \
		-v "$(CURDIR)/k6:/scripts:ro" \
		$(K6_IMAGE) run /scripts/loadtest.js

stress:
	docker run --rm \
		-e BASE_URL=$(BASE_URL) \
		-e MICRO_ADMIN_USER=$(ADMIN_USER) \
		-e MICRO_ADMIN_PASSWORD=$(ADMIN_PASS) \
		-v "$(CURDIR)/k6:/scripts:ro" \
		$(K6_IMAGE) run /scripts/stress.js
