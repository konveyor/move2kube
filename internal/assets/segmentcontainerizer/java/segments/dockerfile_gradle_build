FROM registry.access.redhat.com/ubi8/ubi:latest AS build_base
RUN yum install -y java-1.8.0-openjdk-devel
RUN yum install -y wget
RUN yum install -y unzip
RUN wget https://services.gradle.org/distributions/gradle-6.6-bin.zip -P /tmp
RUN unzip -d /opt/gradle /tmp/gradle-6.6-bin.zip
ENV PATH="$PATH:/opt/gradle/gradle-6.6/bin/"
COPY . /{{ .app_name }}
WORKDIR /{{ .app_name }}
RUN gradle build