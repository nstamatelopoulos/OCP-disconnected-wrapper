FROM golang:1.22.1-bullseye

RUN apt-get update && apt-get install -y jq

WORKDIR /app

RUN groupadd -g 1000 ec2-user && \
    useradd -u 1000 -g 1000 -m -d /home/ec2-user -s /bin/bash ec2-user

COPY . .

RUN go mod download

RUN go build -o /agent

RUN chown -R ec2-user:ec2-user /app

USER 1000:1000

EXPOSE 8090

ENTRYPOINT ["/agent"]