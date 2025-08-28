FROM scratch

# Add ca-certificates for HTTPS requests
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY similarity-go /usr/local/bin/similarity-go

# Set the working directory
WORKDIR /workspace

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/similarity-go"]