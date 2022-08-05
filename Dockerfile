FROM golang:1.17 as build-env

ENV APP_NAME webhook-controller
RUN mkdir -p $GOPATH/src/github.com/shapeblue/$APP_NAME
# Copy application data into image
WORKDIR $GOPATH/src/github.com/shapeblue/$APP_NAME
COPY . .
RUN CGO_ENABLED=0 go build -v -o /$APP_NAME

FROM alpine:3.14
 
# Set environment variable
ENV APP_NAME webhook-controller
 
# Copy only required data into this image
COPY --from=build-env /go/src/github.com/shapeblue/$APP_NAME/commands.json .
COPY --from=build-env /$APP_NAME .
 
# Expose application port
EXPOSE 8089
 
# Start app
CMD ./$APP_NAME
