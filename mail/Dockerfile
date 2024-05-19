FROM golang:latest AS build

WORKDIR /app
COPY ./mail/ ./mail/
COPY ./shared ./shared/
RUN cd ./mail/ && go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o mail .

FROM scratch
WORKDIR app
COPY --from=build /app/mail/mail ./mail
CMD /app/mail
