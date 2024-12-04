# Dockerfile

# Stage 1: Build the Rust application
FROM rust:1.83 as builder

# Create a new directory for the application
WORKDIR /app

# Copy Cargo files to leverage Docker cache
COPY Cargo.toml Cargo.lock ./

# Create an empty main.rs and build dependencies
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release

# Copy the source code
COPY src ./src

# Build the application
RUN cargo build --release

# Stage 2: Create a minimal runtime image
FROM debian:buster-slim

# Copy the compiled binary from the builder stage
COPY --from=builder /app/target/release/telegram_hermes /usr/local/bin/telegram_hermes

# Expose the port the app runs on
EXPOSE 10000

# Run the application
ENTRYPOINT ["/usr/local/bin/telegram_hermes"]
