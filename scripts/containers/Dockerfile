FROM golang:1.20-bullseye AS build

RUN apt update && apt install jq build-essential -y

WORKDIR /beezee
COPY . /beezee

RUN make build

FROM debian:bullseye AS run
COPY ./scripts/containers/build-wrapper.sh /beezee/build-wrapper.sh

VOLUME /beezee
COPY --from=build /beezee/ /beezee/
WORKDIR /beezee

EXPOSE 26656 26657
ENTRYPOINT ["/beezee/build-wrapper.sh"]
CMD ["start", "--log_format", "plain"]
STOPSIGNAL SIGTERM
