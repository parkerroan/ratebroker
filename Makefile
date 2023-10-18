# Makefile

# Define variables
IMAGE_NAME ?= local-images/exampleweb
VERSION ?= 1.0.0

# Docker Compose build
build:
	echo "Building the 'web' service..."
	docker build -t $(IMAGE_NAME):$(VERSION) .

redeploy:
	make build
	echo "Redeploying the 'web' service..."
	docker stack rm exampleweb
	sleep 5
	docker stack deploy -c docker-compose.yml exampleweb

loadtest:
	echo "Running load test..."
	locust -f locust.py --headless -u 1 -r 1 --run-time 5m --csv=output