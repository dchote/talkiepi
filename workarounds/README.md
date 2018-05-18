# Workaround packages for the Raspberry Pi Zero 

Turns out that the Raspberry Pi Zero CPU lacks ARM NEON extensions for hardware floating point, and this breaks the openal packages that ship with standard Raspbian.  These packages are built without NEON support enabled.


Install by `cd talkiepi/workarounds` and then `dpkg -i *`
