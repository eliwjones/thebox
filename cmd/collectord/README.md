Install
=======
```
$ cd cmd/collectord
$ go install
```

Cron (for 'collect' and 'clean')
================================
```
# m h  dom mon dow   command
* * * * * <go_bin>/collectord -root_dir=<dir> -action=collect
0 22 * * * <go_bin>/collectord -root_dir=<dir> -action=clean -yymmdd=yesterday
```
