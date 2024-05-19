FROM golang:latest AS build

WORKDIR /app

COPY ./consumer/ ./consumer/
COPY ./shared ./shared/
RUN cd ./consumer/ && go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o consumer .

FROM scratch
ENV PORT 8000
WORKDIR app
COPY --from=build /app/consumer/consumer ./consumer
EXPOSE $PORT
CMD /app/consumer
