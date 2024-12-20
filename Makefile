include .env
# Variables
DOCKER_TAG=latest
PLATFORM=linux/amd64

.PHONY: run

run-checker: env
	@echo "Starting to handle action, (after then, slack bot server will start to receive request)"
	go run -tags local github.com/b-harvest/Harvestmon/checker --config $(CHECKER_CONFIG)

run-monitor: env
	@echo "Starting to monitor"
	go run github.com/b-harvest/Harvestmon/monitor --config $(MONITOR_CONFIG)


# Build with BuildKit (docker buildx)
.PHONY: buildx

buildx: buildx-checker buildx-monitor

buildx-checker: env
	@echo "Building Docker image for checker with BuildKit..."; \
	if [ -z "$(CHECKER_IMAGE)" ]; then \
		echo "No CHECKER_IMAGE found."; \
		echo "Please enter CHECKER_IMAGE (e.g., 123456789012.dkr.ecr.us-west-1.amazonaws.com/checker):"; \
		read -p "> " DOCKER_IMAGE_NAME_INPUT; \
		docker buildx build --platform=$(PLATFORM) -t $(DOCKER_IMAGE_NAME_INPUT):$(DOCKER_TAG) --provenance=false -f ./checker/Dockerfile --push ./checker/; \
	else \
	  	echo "DOCKER_IMAGE_NAME: $(CHECKER_IMAGE)"; \
		docker buildx build --platform=$(PLATFORM) -t $(CHECKER_IMAGE):$(DOCKER_TAG) --provenance=false -f ./checker/Dockerfile --push ./checker/; \
	fi



buildx-monitor: env
	@echo "Building Docker image for monitor with BuildKit...\n"; \
	if [ -z "$(MONITOR_IMAGE)" ]; then \
		echo "No MONITOR_IMAGE found."; \
		echo "Please enter DOCKER_IMAGE_NAME (e.g., 123456789012.dkr.ecr.us-west-1.amazonaws.com/monitor):"; \
		read -p "> " DOCKER_IMAGE_NAME_INPUT; \
		docker buildx build --platform=$(PLATFORM) -t $(DOCKER_IMAGE_NAME_INPUT):$(DOCKER_TAG) --provenance=false -f ./monitor/Dockerfile --push ./monitor/; \
	else \
	  	echo "DOCKER_IMAGE_NAME: $(MONITOR_IMAGE)"; \
		docker buildx build --platform=$(PLATFORM) -t $(MONITOR_IMAGE):$(DOCKER_TAG) --provenance=false -f ./monitor/Dockerfile --push ./monitor/; \
	fi


# Clean the built images
.PHONY: clean
clean: env
	@echo "Cleaning Docker images..."
	docker rmi $(CHECKER_IMAGE):$(DOCKER_TAG) $
	docker rmi $(MONITOR_IMAGE):$(DOCKER_TAG)

env:
	@if [ -f .env ]; then \
		echo "Loading environment variables from .env..."; \
		set $(cat .env ) 2>&1 /dev/null; \
	fi;

