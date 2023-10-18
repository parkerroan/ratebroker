# Makefile

# Define variables
IMAGE_NAME ?= local-images/exampleweb
VERSION ?= 1.0.0

# Docker Compose build
build:
	echo "Building the 'web' service..."
	docker build -t $(IMAGE_NAME):$(VERSION) .

restart.web:
	echo "Restarting the 'web' service..."
	docker service update --force exampleweb_web

deploy:
 	docker stack deploy -c docker-compose.yml exampleweb

loadtest:
	echo "Running load test..."
	locust -f locust.py --headless -u 1 -r 1 --run-time 5m --csv=output