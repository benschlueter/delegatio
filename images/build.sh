#!/bin/bash

mkosi -f
qemu-img convert -f raw -O qcow2 image.raw image.qcow2