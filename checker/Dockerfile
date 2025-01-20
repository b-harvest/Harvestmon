FROM golang:1.23 as build
WORKDIR /checker
# Copy dependencies list
COPY go.mod go.sum ./
# Build with optional lambda.norpc tag
COPY . .

ENV GODEBUG=asyncpreemptoff=1
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o main
# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /checker/main ./main
COPY config.toml config.toml

ENTRYPOINT [ "./main" ]