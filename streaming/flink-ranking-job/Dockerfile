FROM maven:3.9-eclipse-temurin-17 AS build
WORKDIR /workspace
COPY pom.xml .
COPY src ./src
RUN mvn -q -DskipTests package

FROM flink:1.19.1-java17
COPY --from=build /workspace/target/flink-ranking-job-1.0.0.jar /opt/flink/usrlib/flink-ranking-job.jar
