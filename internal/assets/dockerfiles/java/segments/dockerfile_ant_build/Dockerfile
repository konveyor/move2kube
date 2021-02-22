
FROM registry.access.redhat.com/ubi8/ubi:latest AS build_base
RUN yum install -y java-1.8.0-openjdk-devel
RUN yum install -y wget
RUN yum install -y unzip
RUN wget https://mirrors.estointernet.in/apache//ant/binaries/apache-ant-1.10.8-bin.zip -P /tmp
RUN unzip -d /opt/apache-ant /tmp/apache-ant-1.10.8-bin.zip
ENV PATH="$PATH:/opt/apache-ant/apache-ant-1.10.8/bin/"
COPY . /{{ .app_name }}
WORKDIR /{{ .app_name }}
RUN {{ .ant_cmd }}