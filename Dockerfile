FROM golang:1.25-alpine AS builder
WORKDIR /opt
RUN apk update && apk add --no-cache make git
COPY . /opt/
RUN go mod download
ARG GIT_TAG
ARG GIT_COMMIT
ARG GIT_COMMIT_DATE
RUN make build GIT_TAG=${GIT_TAG} GIT_COMMIT=${GIT_COMMIT} GIT_COMMIT_DATE=${GIT_COMMIT_DATE}

FROM alpine
WORKDIR /opt
RUN apk update && \
    apk add --no-cache ca-certificates
COPY --from=builder /opt/dwight /opt/dwight
ENTRYPOINT ["dwight"]
