https://git.sr.ht/~phw/go-discid

https://gokrazy.org/development/

https://github.com/opus47/cdparanoia/blob/master/paranoia/cdda_paranoia.h

```bash
cd $HOME
FileName='go1.24.6.linux-arm64.tar.gz'
wget https://dl.google.com/go/$FileName
sudo tar -C /usr/local -xvf $FileName
cat >> ~/.bashrc << 'EOF'
export GOPATH=$HOME/go
export PATH=/usr/local/go/bin:$PATH:$GOPATH/bin
EOF
source ~/.bashrc

sudo apt install git cdparanoia libcdparanoia-dev libasound2-dev
```
