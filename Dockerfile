FROM debian:trixie-slim

# Prevent interactive timezone/keyboard prompts during apt-get.
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update
RUN apt-get install -y openjdk-21-jre-headless
RUN apt-get install -y graphviz
RUN apt-get install -y gnuplot
RUN apt-get install -y wget
RUN apt-get install -y bzip2
RUN apt-get install -y ca-certificates
RUN apt-get install -y git
RUN rm -rf /var/lib/apt/lists/*

ENV MAELSTROM_VERSION=0.2.4
RUN wget -q https://github.com/jepsen-io/maelstrom/releases/download/v${MAELSTROM_VERSION}/maelstrom.tar.bz2 && \
    mkdir -p /opt/maelstrom && \
    tar -xjf maelstrom.tar.bz2 -C /opt/maelstrom --strip-components=1 && \
    rm maelstrom.tar.bz2
ENV PATH="/opt/maelstrom:${PATH}"

WORKDIR /app

CMD ["bash"]
