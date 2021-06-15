FROM alpine:latest
COPY main /main
COPY server.key /server.key
COPY server.pem /server.pem
CMD ["/main"]