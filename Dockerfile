FROM golang:1.20-alpine as builder
WORKDIR /workspace
COPY . .
RUN go mod download
ARG TARGETARCH
ARG GIT_HEAD_COMMIT
ARG GIT_TAG_COMMIT
ARG GIT_LAST_TAG
ARG GIT_MODIFIED
ARG GIT_REPO
ARG BUILD_DATE
RUN cd subst/ && CGO_ENABLED=0 GOOS=linux GO111MODULE=on  go build \
        -gcflags "-N -l" \
        -ldflags "-X main.GitRepo=$GIT_REPO -X main.GitTag=$GIT_LAST_TAG -X main.GitCommit=$GIT_HEAD_COMMIT -X main.GitDirty=$GIT_MODIFIED -X main.BuildTime=$BUILD_DATE" \
        -o subst


FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/subst/subst /subst
USER nonroot:nonroot
ENTRYPOINT ["/subst"]