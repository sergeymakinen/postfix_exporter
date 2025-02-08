FROM quay.io/prometheus/busybox:latest
LABEL maintainer="Sergey Makinen <sergey@makinen.ru>"

ARG TARGETOS
ARG TARGETARCH
COPY dist/postfix_exporter_${TARGETOS}_${TARGETARCH}/postfix_exporter /bin/postfix_exporter

EXPOSE 9907
USER nobody
ENTRYPOINT ["/bin/postfix_exporter"]
