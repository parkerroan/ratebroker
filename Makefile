# Makefile

# Define variables
IMAGE_NAME ?= local-images/exampleweb
VERSION ?= 1.0.0

# Docker Compose build
build:
	echo "Building the 'web' service..."
	docker build -t $(IMAGE_NAME):$(VERSION) .

