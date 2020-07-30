# sshz



Parallel ssh command execution.



Usage:

```shell
$ echo "127.0.1.2\n127.0.0.1"  | ./sshz -u $USER id
127.0.1.2:22 uid=1000(user) gid=1000(user) groups=1000(user)
127.0.0.1:22 uid=1000(user) gid=1000(user) groups=1000(user)
```



The number of commands can be arbitrary:

```shell
$ echo "127.0.1.2\n127.0.0.1"  | ./sshz -u $USER 'echo 1' 'echo 2'
127.0.0.1:22 1
127.0.0.1:22 2
127.0.1.2:22 1
127.0.1.2:22 2
```



By default, both stderr and stdout are captured, and both streams can be identified when the `extended` output format is enabled:

```shell
$ echo "127.0.1.2\n127.0.0.1"  | ./sshz -u $USER --output-format=extended\
        'echo this should appear in stdout'\
        'echo this hould appear in stderr 1>&2'
127.0.0.1:22 0 stdout this should appear in stdout
127.0.0.1:22 1 stderr this hould appear in stderr
127.0.1.2:22 0 stdout this should appear in stdout
127.0.1.2:22 1 stderr this hould appear in stderr
```

Alternatively, stderr can just be ignored with the `--ignore-stderr` option:

```shell
$ echo "127.0.1.2\n127.0.0.1"  | ./sshz -u $USER --output-format=extended  \
        --ignore-stderr 'echo this should appear in stdout'\
        'echo this hould appear in stderr 1>&2'
127.0.1.2:22 0 stdout this should appear in stdout
127.0.0.1:22 0 stdout this should appear in stdout
```

The hosts list can be a file specified with `-l` or `--hosts-list`. Another available `output-format` is `json`:
```shell
$ sshz -l <(echo localhost) -u $USER --output-format json 'echo hello' | jq
[
  {
    "Host": "localhost:22",
    "Output": [
      {
        "Stream": "stdout",
        "Line": "hello",
        "Timestamp": 4
      }
    ],
    "ExecutionTimeMicros": 4,
    "SeqNum": 0
  }
]
```

When multiple commands are specified, the execution (for each host) stops is a command's status code is not zero:
```shell
$ sshz -l <(echo localhost) -u $USER --output-format json \
        'echo should be visible' \
        'stat /non-existing-file' \
        'echo should not be visible' | jq
[
  {
    "Host": "localhost:22",
    "Output": [
      {
        "Stream": "stdout",
        "Line": "should be visible",
        "Timestamp": 5
      }
    ],
    "ExecutionTimeMicros": 5,
    "SeqNum": 0
  },
  {
    "Host": "localhost:22",
    "Output": [
      {
        "Stream": "stderr",
        "Line": "stat: /non-existing-file: stat: No such file or directory",
        "Timestamp": 5
      }
    ],
    "Error": {},
    "ExecutionTimeMicros": 5,
    "SeqNum": 1
  }
]
```



## Concurrency

The number of parallel connections is, by default, 64. This parameter is configurable:

```shell
$ echo "127.0.1.2\n127.0.0.1"  | ./sshz -u $USER --concurrency=1 'sleep 5; date'
127.0.1.2:22 Thu 30 Jul 2020 02:57:38 PM CEST
127.0.0.1:22 Thu 30 Jul 2020 02:57:44 PM CEST
```

```shell
$ echo "127.0.1.2\n127.0.0.1"  | ./sshz -u $USER 'sleep 5; date'
127.0.1.2:22 Thu 30 Jul 2020 02:58:38 PM CEST
127.0.0.1:22 Thu 30 Jul 2020 02:58:38 PM CEST
```

