#####################
### Server Build Step
#####################
FROM --platform=${TARGETPLATFORM:-linux/amd64} golang:1.22-bookworm AS server-builder 

RUN apt-get update && apt-get install -y \
    build-essential \
    gcc \
    libc6-dev

RUN mkdir /server
WORKDIR /server

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG COMMITHASH
ARG VERSION
RUN COMMITHASH=${COMMITHASH} VERSION=${VERSION} GOOS=${TARGETOS} GOARCH=${TARGETARCH} make build-server
################
### Runtime Step
################
FROM --platform=${TARGETPLATFORM:-linux/amd64} debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates

WORKDIR /app

COPY --from=server-builder /server/migrations /app/migrations
COPY --from=server-builder /server/views /app/views
COPY --from=server-builder /server/locale /app/locale
COPY --from=server-builder /server/LICENSE /app
COPY --from=server-builder /server/fider /app

# Frontend is served independently - see fider-frontend/Dockerfile

EXPOSE 3000

HEALTHCHECK --timeout=5s CMD ./fider ping

CMD ./fider migrate && ./fider