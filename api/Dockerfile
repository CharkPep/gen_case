FROM golang:latest AS build

WORKDIR /app

COPY ./api/ ./api/
COPY ./shared ./shared/
RUN cd ./api/ && go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o api .

FROM scratch
ENV PORT 8000
WORKDIR app
COPY --from=build /app/api/api ./api
EXPOSE $PORT
ENTRYPOINT /app/api
 
