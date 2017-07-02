# Boot to talkiepi
![assembled1](talkiepi_assembled_1.jpg "Assembled talkiepi 1")

This is a simple overview to scratch install talkiepi on your Raspberry Pi, and have it start on boot.  This guide assumes you are using a Raspberry Pi Zero W and the Plugable USB adapter (http://plugable.com/products/usb-audio/).


By default talkiepi will run without any arguments, it will autogenerate a username and then connect to my mumble server.
You can change this behavior by appending commandline arguments of `-server YOUR_SERVER_ADDRESS`, `-username YOUR_USERNAME` to the ExecStart line in `/etc/systemd/system/mumble.service` once installed.

talkiepi will also accept arguments for `-password`, `-insecure`, `-certificate` and `-channel`, all defined in `cmd/talkiepi/main.go`, if you run your own mumble server, these will be self explanatory.

## Flash Raspbian Jessie, set up wifi, etc.
https://www.losant.com/blog/getting-started-with-the-raspberry-pi-zero-w-without-a-monitor

## Create a user

As root on your Raspberry Pi (`sudo -i`), create a mumble user:
```
adduser --disabled-password --disabled-login --gecos "" mumble
usermod -a -G cdrom,audio,video,plugdev,users,dialout,dip,input,gpio mumble
```

## Install

As root on your Raspberry Pi (`sudo -i`), install golang and other required dependencies, then build talkiepi:
```
apt-get install golang libopenal-dev libopus-dev git

su mumble

mkdir ~/gocode
mkdir ~/bin

export GOPATH=/home/mumble/gocode
export GOBIN=/home/mumble/bin

cd $GOPATH

go get github.com/layeh/gopus
go get github.com/dchote/talkiepi

cd $GOPATH/src/github.com/dchote/talkiepi

git remote add fork https://github.com/WilliamLiska/talkiepi.git
git pull fork add-volume-controls
git checkout add-volume-controls

go build -o /home/mumble/bin/talkiepi cmd/talkiepi/main.go 
```


## Start on boot

As root on your Raspberry Pi (`sudo -i`), copy mumble.service in to place:
```
cp /home/mumble/gocode/src/github.com/dchote/talkiepi/conf/systemd/mumble.service /etc/systemd/system/mumble.service
```

Update /etc/systemd/system.mumble.service using `sudo nano /etc/systemd/system/mumble.service`, appending `-server [serverip:port] -username [username] -password [password]` to `ExecStart = /home/mumble/bin/talkiepi`

Enable the service to run at boot:
```
systemctl enable mumble.service
```

## Create a certificate

This is optional, mainly if you want to register your talkiepi against a mumble server and apply ACLs.
```
su mumble
cd ~

openssl genrsa -aes256 -out key.pem
```

Enter a simple passphrase, its ok, we will remove it shortly...

```
openssl req -new -x509 -key key.pem -out cert.pem -days 1095
```

Enter your passphrase again, and fill out the certificate info as much as you like, its not really that important if you're just hacking around with this.

```
openssl rsa -in key.pem -out nopasskey.pem
```

Enter your password for the last time.

```
cat nopasskey.pem cert.pem > mumble.pem
```

Now as root again (`sudo -i`), edit `/etc/systemd/system/mumble.service` appending `-username USERNAME_TO_REGISTER -certificate /home/mumble/mumble.pem` at the end of `ExecStart = /home/mumble/bin/talkiepi`

Run `systemctl daemon-reload` and then `service mumble restart` and you should be set with a tls certificate!


## Use your USB speakerphone

If you are using a USB speakerphone such as the US Robotics one that I am using, you will need to change the default system sound device.
As root on your Raspberry Pi (`sudo -i`), find your device by running `aplay -l`, take note of the index of the device (likely 1) and then edit the alsa config (`/usr/share/alsa/alsa.conf`), changing the following:
```
defaults.ctl.card 1
defaults.pcm.card 1
```
_1 being the index of your device_


If your speakerphone is too quiet, you can adjust the volume using amixer as such:
```
amixer -c 1 set Headphone 60%
```
_1 being the index of your device_

## Install raspberry-wifi-conf

In order to support SSH-free Wifi configuration and pushbutton Wifi reconfiguration, install this in /home/mumble/raspberry-wifi-conf: https://github.com/WilliamLiska/raspberry-wifi-conf

## Install supertalkie-buttons

supertalkie-buttons is a python script that watches for miscellaneous button presses and runs scripts.  Currently it's configured to kick off raspberry-wifi-conf in ForceChange mode to 'reset' the Wifi settings.  Install it from https://github.com/WilliamLiska/supertalkie-buttons.

I will be adding volume control settings in an upcoming push.
