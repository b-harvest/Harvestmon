FROM golang:1.22.4 as build
WORKDIR /checker
# Copy dependencies list
COPY go.mod go.sum ./
# Build with optional lambda.norpc tag
COPY . .
RUN go mod tidy
RUN go build -tags lambda.norpc -o main
# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /checker/main ./main
COPY config.toml config.toml

ENTRYPOINT [ "./main" ]