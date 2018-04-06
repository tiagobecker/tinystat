FROM alpine:latest
RUN apk add --no-cache ca-certificates
ADD tinystat /usr/local/bin/tinystat
ADD /web /web/
EXPOSE 8080
CMD tinystat