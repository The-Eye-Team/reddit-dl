# reddit-dl
![loc](https://sloc.xyz/github/The-Eye-Team/reddit-dl)
[![license](https://img.shields.io/github/license/The-Eye-Team/reddit-dl.svg)](https://github.com/The-Eye-Team/reddit-dl/blob/master/LICENSE)
[![discord](https://img.shields.io/discord/302796547656253441.svg)](https://discord.gg/the-eye)
[![circleci](https://circleci.com/gh/The-Eye-Team/reddit-dl.svg?style=svg)](https://circleci.com/gh/The-Eye-Team/reddit-dl)
[![release](https://img.shields.io/github/v/release/The-Eye-Team/reddit-dl)](https://github.com/The-Eye-Team/reddit-dl/releases/latest)
[![goreportcard](https://goreportcard.com/badge/github.com/The-Eye-Team/reddit-dl)](https://goreportcard.com/report/github.com/The-Eye-Team/reddit-dl)
[![codefactor](https://www.codefactor.io/repository/github/The-Eye-Team/reddit-dl/badge)](https://www.codefactor.io/repository/github/The-Eye-Team/reddit-dl)

Downloader for submissions to reddit.com. Supports both subreddits and users.

## Download
https://github.com/The-Eye-Team/reddit-dl/releases/latest

## Usage
```sh
    --do-comments       Enable this flag to save post comments.
    --concurrency int   Maximum number of simultaneous downloads. (default 10)
-d, --domain string     The host of a domain to archive.
    --mbpp-bar-gradient Enabling this will make the bar gradient from red/yellow/green.
    --no-domain-dir     Enable this flag to disable adding 'reddit.com' to --save-dir.
    --no-pics           Enable this flag to disable the saving of post attachments.
    --save-dir string   Path to a directory to save to.
-r, --subreddit string  The name of a subreddit to archive. (ex. AskReddit, unixporn, CasualConversation, etc.)
-u, --user string       The name of a user to archive. (ex. spez, PoppinKREAM, Shitty_Watercolour, etc.)
```
The flags `-r` and `-u` may be passed multiple times to download many reddits at once.

## License
MIT
