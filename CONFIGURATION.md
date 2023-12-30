# Postfix exporter configuration

The file is written in [YAML format](http://en.wikipedia.org/wiki/YAML), defined by the scheme described below.
Brackets indicate that a parameter is optional.
For non-list parameters the value is set to the specified default.

Generic placeholders are defined as follows:

* `<string>`: a regular string
* `<regex>`: a regular expression (see https://golang.org/s/re2syntax)

The other placeholders are specified separately.

See [postfix.yml](exporter/testdata/postfix.yml) for configuration examples.

```yml
status_replies:
  [ - <status_reply>, ... ]
smtp_replies:
  [ - <smtp_reply>, ... ]
noqueue_reject_replies:
  [ - <noqueue_reject_reply>, ... ]
```

### `<status_reply>`

Example log entry:

```
Jan 1 00:00:00 hostname postfix/smtp[12345]: 123456789AB: to=<user@example.com>, relay=example.com[123.45.67.89]:25, delay=1.23, delays=1.23/1.23/1.23/1.23, dsn=1.2.3, status=bounced (host example.com[123.45.67.89] said: 123 #1.2.3 Reasons (in reply to end of DATA command))
```

In this case:

* `123` is a status code
* `1.2.3` is an enhanced status code
* `Reasons` is the text of the reply

```yml
# Only allow specific statuses.
statuses:
  [ - <string>, ... ]

# The regular expression matching the reply code, enhanced code or text.
regexp: <regex>

# Match type. Accepted values: code, enhanced_code, text.
[ match: <string> | default = "text" ]

# The replacement text (may include placeholders supported by Go, see https://pkg.go.dev/regexp#Regexp.Expand).
text: <string>
```

### `<smtp_reply>`

Example log entry:

```
Jan 1 00:00:00 hostname postfix/smtp[12345]: 123456789AB: host example.com[123.45.67.89] said: 123 1.2.3 Reasons (in reply to RCPT TO command)
```

In this case:

* `123` is a status code
* `1.2.3` is an enhanced status code
* `Reasons` is the text of the reply

```yml
# The regular expression matching the reply code, enhanced code or text.
regexp: <regex>

# Match type. Accepted values: code, enhanced_code, text.
[ match: <string> | default = "text" ]

# The replacement text (may include placeholders supported by Go, see https://pkg.go.dev/regexp#Regexp.Expand).
text: <string>
```

### `<noqueue_reject_reply>`

Example log entry:

```
Jan 1 00:00:00 hostname postfix/smtpd[12345]: NOQUEUE: reject: RCPT from example.com[123.45.67.89]: 123 1.2.3 <user@example.com>: Reasons; from=<user@example.com> to=<user@example.com> proto=ESMTP helo=<example.com>
```

In this case:

* `123` is a status code
* `1.2.3` is an enhanced status code
* `Reasons` is the text of the reply

```yml
# The regular expression matching the reply code, enhanced code or text.
regexp: <regex>

# Match type. Accepted values: code, enhanced_code, text.
[ match: <string> | default = "text" ]

# The replacement text (may include placeholders supported by Go, see https://pkg.go.dev/regexp#Regexp.Expand).
text: <string>
```
