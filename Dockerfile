FROM golang AS compiling_stage
RUN mkdir -p /go/src/pipeline
WORKDIR /go/src/pipeline
ADD main.go .
ADD go.mod .
RUN CGO_ENABLED=0 GOOS=linux go build .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=compiling_stage /go/src/pipeline .
ENTRYPOINT ./pipeline