FROM alpine:latest
ARG TARGETPLATFORM
ENV USER=micro
ENV GROUPNAME=$USER
ARG UID=1001
ARG GID=1001
RUN addgroup --gid "$GID" "$GROUPNAME" \
    && adduser \
    --disabled-password \
    --gecos "" \
    --home "/micro" \
    --ingroup "$GROUPNAME" \
    --no-create-home \
    --uid "$UID" "$USER"

ENV PATH=/usr/local/go/bin:$PATH
RUN apk --no-cache add git make curl
COPY --from=golang:1.26.0-alpine /usr/local/go /usr/local/go

COPY $TARGETPLATFORM/micro /usr/local/go/bin/
COPY $TARGETPLATFORM/protoc-gen-micro /usr/local/go/bin/

WORKDIR /micro
EXPOSE 8080
ENTRYPOINT ["/usr/local/go/bin/micro"]
CMD ["server"]
