# gmi
Yet-another Gemini reader

1. One program built in Tcell which runs in the terminal:
![remote ssh session](/cmd/term/tc01.png)

2. Another built in Ebiten which hopefully becomes my Android app:
![mobile 360x640](/cmd/mobile/eb03.png) 

## Quickstart
```bash
git clone https://github.com/shrmpy/gmi
cd gmi && go build -o test cmd/term/*.go
./test
```
## Build in Local Container
```bash
cd gmi
docker build -t bc .
docker run -ti --rm --entrypoint sh -v $PWD:/opt/test bc
cp -R /opt/test/cmd/mobile/*.go cmd/mobile/
go build -o test cmd/mobile/*.go
cp test /opt/test/testmo
exit
./testmo
```
## Make your own snap package
[![gmird](https://snapcraft.io/gmird/badge.svg)](https://snapcraft.io/gmird)
```bash
# ub server includes a empty lxd?
sudo snap remove --purge lxd
# reinstall lxd
sudo snap install lxd
sudo lxd init --auto
sudo usermod -a -G lxd ${USER}
# view config
lxc version
lxc profile show default
lxc storage show default
echo 'export SNAPCRAFT_BUILD_ENVIRONMENT=lxd' >> ~/.profile
sudo reboot
# retrieve YAML 
git clone https://gitlab.com/shrmpy/gmi.git
cd gmi
# make snap 
snapcraft
# local install
sudo snap install gmird_0.0.14_arm64.snap --dangerous
# start reader
gmird.mobile
```

## Credits
Font Renderer
 by [tinne26](https://github.com/tinne26/etxt) ([LICENSE](https://github.com/tinne26/etxt/blob/main/LICENSE))

Golang Gemini Demo 
 by [Solderpunk](https://tildegit.org/solderpunk/gemini-demo-3) ([LICENSE](https://tildegit.org/solderpunk/gemini-demo-3/src/branch/master/LICENSE))

min Gemini browser 
 by [Adrian Hesketh](https://github.com/a-h/min) ([LICENSE](https://github.com/a-h/min/blob/master/LICENSE))

Tcell by [Garrett D'Amore](https://github.com/gdamore/tcell/) ([LICENSE](https://github.com/gdamore/tcell/blob/master/LICENSE))

Ebiten by [Hajime Hoshi](https://github.com/hajimehoshi/ebiten/) ([LICENSE](https://github.com/hajimehoshi/ebiten/blob/main/LICENSE))

Lexical Scanning in Go
 by [Rob Pike](https://go.dev/blog/sydney-gtug)
 [template source](https://go.dev/src/text/template/parse/lex.go) ([LICENSE](https://github.com/golang/go/blob/master/LICENSE))

Noto Sans Mono
 by [Google](https://fonts.google.com/noto/specimen/Noto+Sans+Mono/about) ([LICENSE](https://scripts.sil.org/cms/scripts/page.php?site_id=nrsi&id=OFL))

DejaVu Sans Mono
 by [DejaVu](https://dejavu-fonts.github.io/) ([LICENSE](https://github.com/dejavu-fonts/dejavu-fonts/blob/master/LICENSE))

