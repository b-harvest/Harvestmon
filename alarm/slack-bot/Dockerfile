FROM golang:1.22.4 as build
WORKDIR /slack-bot
# Copy dependencies list
COPY go.mod go.sum ./
# Build with optional lambda.norpc tag
COPY main.go .

RUN go mod tidy
RUN go build -tags lambda.norpc -o main
# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /slack-bot/main ./main
ENTRYPOINT [ "./main" ]