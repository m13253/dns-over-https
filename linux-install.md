# Ubuntu Install
> Tested on a clean install of `Ubuntu 16.04 LTS`

## Intalling go
Install `Go >= 1.9`

```bash
sudo apt update
sudo apt install golang-1.10 -y
```

Add the newly install `go` to the path

```bash
export PATH=$PATH:/usr/lib/go-1.10/bin
```

Test to make sure that you can execute `go`

```bash
go version
```
which should output something like

```bash
go version go1.10.1 linux/amd64
```

## Installing dns-over-https

Clone this repo


```bash
git clone https://github.com/m13253/dns-over-https.git
```

Change directory to the cloned repo

```bash
cd dns-over-https
```

make and install

```bash
make
sudo make install
```	


