FROM rust:1.83 AS build

# create a new empty shell project
WORKDIR /build
RUN USER=root cargo new --bin telegram-hermes

# copy over your manifests
COPY Cargo.lock Cargo.toml ./telegram-hermes/

WORKDIR /build/telegram-hermes

# this build step will cache your dependencies
RUN cargo build --release && rm src/*.rs && rm target/release/deps/telegram_hermes*

# copy your source tree
COPY ./src/* src/

# build for release
RUN cargo build --release

# our final base
FROM debian:bookworm-slim

RUN apt-get update && apt install -y ca-certificates

# copy the build artifact from the build stage
COPY --from=build /build/telegram-hermes/target/release/telegram-hermes .

EXPOSE 10000

# set the startup command to run your binary
CMD ["./telegram-hermes"]