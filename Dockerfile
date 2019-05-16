FROM golang:alpine AS build-env

WORKDIR /src
ADD . /src
RUN apk add --no-cache git make
RUN make

FROM alpine:latest

COPY --from=build-env /src/doh-client/doh-client /doh-client
COPY --from=build-env /src/doh-server/doh-server /doh-server

ADD doh-client/doh-client.conf /doh-client.conf
ADD doh-server/doh-server.conf /doh-server.conf

RUN sed -i '$!N;s/"127.0.0.1:8053",\s*"\[::1\]:8053",/":8053",/;P;D' /doh-server.conf

EXPOSE 8053

ENTRYPOINT ["/doh-server"]
CMD ["-conf", "/doh-server.conf"]
