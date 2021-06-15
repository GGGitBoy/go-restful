FROM alpine:3
COPY go-restful go-restful
CMD ["/pdf-sender"]