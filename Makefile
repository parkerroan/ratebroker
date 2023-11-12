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
	-(cd locust && locust -f locust.py --headless -u 500 -r .2 --run-time 5m)
	-(cd locust && venv/bin/python plot.py log.csv)

test.integration:
	go test -timeout 30s -run Integration github.com/parkerroan/ratebroker -count=1 -v

test.unit:
	go test -timeout 30s -run Unit github.com/parkerroan/ratebroker -count=1 -v
	go test -timeout 30s -run Unit github.com/parkerroan/ratebroker/limiter -count=1 -v

benchmarks.integration:
	go test -timeout 30s -bench=Integration github.com/parkerroan/ratebroker -run=^$$ -count=1 -v -benchmem
	go test -timeout 30s -bench=Integration github.com/parkerroan/ratebroker/limiter -run=^$$ -count=1 -v -benchmem

benchmarks.unit:
	go test -timeout 30s -bench=Unit github.com/parkerroan/ratebroker -run=^$$ -count=1 -v
	go test -timeout 30s -bench=Unit github.com/parkerroan/ratebroker/limiter -run=^$$ -count=1 -v
