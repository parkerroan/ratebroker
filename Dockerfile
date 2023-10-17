# Start from the desired base image
FROM golang:1.21 

# Add the source code
ADD . /src

# Set the working directory
WORKDIR /src

# Build the application
RUN go build -o app cmd/exampleweb/main.go 

# Set the binary as executable
RUN chmod +x ./app

# Run the binary
CMD ["./app"]

