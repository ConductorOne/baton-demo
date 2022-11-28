FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-demo"]
COPY baton-demo /