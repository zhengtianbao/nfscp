## Overview

nfscp is a tool to transfer files between host and a NFS(Network File System) directly without mount.

## Features

- Copy file from local to NFS
- Copy file from NFS to local
- Recursively copy directory from local to NFS
- Recursively copy directory from NFS to local
- Transfer speed limit

## Installation

To install type (in the folder that contains the Makefile):

```
make build
```

The target binary will be at `_output/bin/$ARCH/nfscp`.

## Example usage

Copy local directory to remote NFS server with speed limit 10KB/s:

```
nfscp -l 10 -r /path/to/dir 192.168.1.100:/nfs/
```

Copy remote NFS file **hello.txt** to local under **/tmp** directory:

```
nfscp 192.168.1.100:/nfs/hello.txt /tmp
```

## Help

If you need any help, feel free to open a [new Issue](https://github.com/zhengtianbao/nfscp/issues/new).
