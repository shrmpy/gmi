
FROM golang:1.18

RUN apt update && apt install -y --no-install-recommends \
    libc6-dev libglu1-mesa-dev libgl1-mesa-dev libxcursor-dev \
    libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config

WORKDIR /app
COPY . .
RUN go get github.com/hajimehoshi/ebiten/v2

##RUN go build cmd/mobile/*.go
RUN go get github.com/gdamore/tcell/v2
##RUN go build -o hello cmd/term/*.go

RUN go get github.com/tinne26/etxt


