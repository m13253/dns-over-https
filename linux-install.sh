#!/bin/bash

# See the linux-install.md (README) first. 
set -e 

sudo apt update
sudo apt install golang-1.10 git -y
export GOHOME=/usr/lib/go-1.10/
cd /tmp
git clone https://github.com/m13253/dns-over-https.git
cd dns-over-https
make 
sudo make install
