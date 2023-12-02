FROM alpine:latest

ENV TZ="Asia/Shanghai"

ARG TARGETOS
ARG TARGETARCH

COPY ./script/entrypoint.sh /entrypoint.sh

RUN apk update && apk add ca-certificates tzdata && \
    update-ca-certificates && \
    ln -fs /usr/share/zoneinfo/$TZ /etc/localtime && \
    chmod +x /entrypoint.sh

WORKDIR /dashboard
COPY dist/dashboard-${TARGETOS}-${TARGETARCH} ./app

VOLUME ["/dashboard/data"]
EXPOSE 80 5555
ENTRYPOINT ["/entrypoint.sh"]