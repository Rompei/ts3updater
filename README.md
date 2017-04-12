# ts3updater

ts3updater updates TeamSpeak3 server on Docker container automatically.


## Requirements

Deploy docker container with [andreasheil/docker-teamspeak3](https://github.com/andreasheil/docker-teamspeak3)

## Usage 

Set this program on cron or something to schedule jobs.

```
Usage of ./ts3updater:
  -b string
    	Backup directory.
  -c string
    	Container name. (default "ts3-server")
  -d string
    	Data directory on host.
  -n string
    	Slack notification url.
```
