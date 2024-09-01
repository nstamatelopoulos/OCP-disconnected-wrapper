FROM registry.access.redhat.com/ubi9/go-toolset as builder

WORKDIR /app

RUN chmod -R a+rwx /app

COPY . .

RUN go mod download

RUN go build -o agent

FROM registry.access.redhat.com/ubi9/ubi-minimal

# Install required packages

RUN microdnf update -y && \
    microdnf install -y jq hostname openssh wget tar unzip

# Install AWS CLI v2
RUN wget "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -O "awscliv2.zip" && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -rf awscliv2.zip aws

# Mount the user in the container
RUN groupadd -g 1000 ec2-user && \
    useradd -u 1000 -g 1000 -m -d /home/ec2-user -s /bin/bash ec2-user

COPY --from=builder /app/agent /app/agent

COPY cluster-installation-script.sh.template /app

RUN chown -R ec2-user:ec2-user /app

USER 1000:1000

EXPOSE 8090

ENTRYPOINT ["/app/agent"]