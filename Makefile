# Makefile

# Define variables
IMAGE_NAME ?= local-images/exampleweb
VERSION ?= 1.0.0

# Docker Compose build
build:
	echo "Building the 'web' service..."
	docker build -t $(IMAGE_NAME):$(VERSION) .

swarm.restart.web:
	echo "Restarting the 'web' service..."
	docker service update --force exampleweb_web

swarm.deploy: 
	docker stack deploy -c docker-compose.yml exampleweb

swarm.down: 
	docker stack rm exampleweb

loadtest:
	echo "Running load test..."
	-rm -f locust/log.csv
	-(cd locust && locust -f locust.py --headless -u 100 -r 1 --run-time 5m)
	-(cd locust && venv/bin/python plot.py log.csv)
	


