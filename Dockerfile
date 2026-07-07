# atomos — ECM (Eidolons Context Management) compose/verify executor MCP.
# Thin 2-stage build: a golang:1.23 builder produces a fully-static CGO-off
# binary; the runtime layer is gcr.io/distroless/static-debian12:nonroot
# (CA certs + a nonroot user, a few MB). Mirrors tonberry's build shape —
# the ecosystem's stateless-executor precedent (ADR §1).

# ---- builder ----
FROM golang:1.23 AS builder
WORKDIR /src

# Module graph first for layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Source.
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Fully-static, stripped, reproducible-ish binary.
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -trimpath -o /out/atomos ./cmd/atomos

# ---- runtime ----
# distroless/static: no shell, no package manager, no network client linked
# beyond what the Go binary itself statically embeds (none — atomos has no
# HTTP client). Structural capability starvation (README.md "Fence").
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/atomos /atomos
# Default workdir is the mounted project tree (the future nexus MCP template
# mounts the consumer project at /workspace); atomos writes only
# .eidolons/.context/ there when write_sidecar is true.
WORKDIR /workspace
ENTRYPOINT ["/atomos"]
CMD ["serve"]
