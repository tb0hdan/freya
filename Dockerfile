FROM golang:alpine AS build-env
WORKDIR /
ADD . /
RUN apk update
RUN apk add gcc git make musl-dev xz
RUN apk add --no-cache ca-certificates apache2-utils
RUN git clone https://github.com/blechschmidt/massdns
RUN cd massdns; make
RUN make freya


# final stage
FROM alpine
WORKDIR /
COPY --from=build-env /etc/ssl /etc/ssl
COPY --from=build-env /freya /
COPY --from=build-env /usr/bin/xz /usr/bin/
COPY --from=build-env /usr/lib/liblzma.so.5 /usr/lib/
COPY --from=build-env /massdns/bin/massdns /
CMD /freya
EXPOSE 80
