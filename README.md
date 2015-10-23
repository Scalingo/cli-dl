# CLI-dl

This repository contains the server hosted at [https://cli-dl.scalingo.io/](https://cli-dl.scalingo.io/) which allows you to download the command-line client on the platform and version you want.

## Endpoints

* `/install` : Display the content of CLI installer shell script.

* `/release/<archive_name>` : archive_name is the exact name of the archive you want to download (it must be part of these : [https://github.com/Scalingo/cli/releases](https://github.com/Scalingo/cli/releases)).

Note: If you want the download link of the latest release, you can replace the version by "latest", i.e:

From:
> /release/scalingo_1.0.0_linux_amd64.tar.gz

To:
> /release/scalingo_**latest**_linux_amd64.tar.gz