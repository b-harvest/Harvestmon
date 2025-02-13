FROM golang:1.23.4 as build
WORKDIR /manager
# Copy dependencies list
COPY go.mod go.sum ./
# Build with optional lambda.norpc tag
COPY . .
RUN go mod tidy
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -tags lambda -o main
# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2
COPY --from=build /manager/main ./main
COPY config.toml config.toml

ENTRYPOINT [ "./main" ]