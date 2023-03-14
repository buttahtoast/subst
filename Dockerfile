FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY subst /subst
USER nonroot:nonroot
ENTRYPOINT ["/subst"]