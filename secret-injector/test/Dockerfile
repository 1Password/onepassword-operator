FROM ubuntu:latest
ARG VERSION

RUN apt-get update && apt-get install -y curl unzip jq && \
    curl -o 1password.zip https://bucket.agilebits.com/cli-private-beta/v2/op_linux_amd64_v2-alpha2.zip && \
    unzip 1password.zip -d /usr/local/bin && \
    rm 1password.zip

CMD ["op"]