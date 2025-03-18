# syntax=docker/dockerfile:1-labs
FROM golang:latest AS base

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY --exclude=entrypoint.py . .

RUN go build -o main cmd/pack3d/main.go

FROM python:latest
USER root
COPY entrypoint.py .
COPY model.stl .
COPY --from=base /app/main .
COPY input.json .
CMD ["python3", "entrypoint.py"]
