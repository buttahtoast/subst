FROM golang:1.19-alpine as builder
WORKDIR /build
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on  go build -o bin ./subst/

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /build/bin /subst
USER nonroot:nonroot
ENTRYPOINT ["/subst"]