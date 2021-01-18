FROM registry.access.redhat.com/ubi8/ubi:latest AS build_base
RUN yum install -y java-1.8.0-openjdk-devel
RUN yum install -y maven
COPY . /{{ .app_name }}
WORKDIR /{{ .app_name }}
RUN mvn package