# Installation

`gribic` is a single binary built for the Linux, Mac OS and Windows platforms distributed via [Github releases](https://github.com/karimra/gribic/releases).

### Linux/Mac OS

To download & install the latest release the following automated [installation script](https://github.com/karimra/gribic/blob/main/install.sh) can be used:

```bash
bash -c "$(curl -sL https://get-gribic.kmrd.dev)"
```

As a result, the latest `gribic` version will be installed in the `/usr/local/bin` directory and the version information will be printed out.

```text
Downloading gribic_0.0.2_darwin_x86_64.tar.gz
Preparing to install gribic 0.0.2 into /usr/local/bin

gribic installed into /usr/local/bin/gribic
version : 0.0.2
 commit : 3bb2670
   date : 2021-05-05T16:39:59Z
 gitURL : https://github.com/karimra/gribic
   docs : https://gribic.kmrd.dev
```

#### Packages

Linux users running distributions with support for `deb`/`rpm` packages can install `gribic` using pre-built packages:

```bash
bash -c "$(curl -sL https://get-gribic.kmrd.dev)" -- --use-pkg
```

### Docker

The `gribic` container image can be pulled from GitHub container registries. The tag of the image corresponds to the release version and `latest` tag points to the latest available release:

```bash
# pull latest release from github registry
docker pull ghcr.io/karimra/gribic:latest
# pull a specific release from github registry
docker pull ghcr.io/karimra/gribic:0.0.1
```

```bash
docker run \
       --network host \
       --rm ghcr.io/karimra/gribic --help
```
