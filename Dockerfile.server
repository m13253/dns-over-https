FROM golang:alpine AS build-env

RUN apk add --no-cache git make

WORKDIR /src
ADD . /src
RUN make doh-server/doh-server

FROM alpine:latest

COPY --from=build-env /src/doh-server/doh-server /doh-server

ADD doh-server/doh-server.conf /doh-server.conf

RUN sed -i '$!N;s/"127.0.0.1:8053",\s*"\[::1\]:8053",/":8053",/;P;D' /doh-server.conf

EXPOSE 8053

ENTRYPOINT ["/doh-server"]
CMD ["-conf", "/doh-server.conf"]
