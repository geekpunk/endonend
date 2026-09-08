# Build context is the repo root (see docker-compose.yml), since this
# module's go.mod replaces endonend/protocol with a local ../protocol path,
# per KB/0006-container-infrastructure.md's Go image build pattern.
FROM golang:1.25.5 AS build
WORKDIR /src
COPY protocol/ ./protocol/
COPY api/ ./api/
WORKDIR /src/api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
ENTRYPOINT ["/server"]
