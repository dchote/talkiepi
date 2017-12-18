# Workaround packages for the Raspberry Pi Zero 
### (or other non-floating point hardware)

Turns out that the Raspberry Pi Zero lacks NEON extensions for hardware floating point, and this breaks the openal packages that ship with standard Raspbian.  These packages are built without NEON support enabled.
