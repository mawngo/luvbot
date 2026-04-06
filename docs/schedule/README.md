# Schedule the bot with Systemd and XVFB

Run the bot hourly without GUI.

Requirements:

- Systemd
- XVFB
- Chromium Browser

## Set up XVFB service

Copy the `xvfb@.service` systemd unit file to `/etc/systemd/system/`:

```shell
sudo cp xvfb@.service /etc/systemd/system/xvfb@.service
```

Enable the service:

```shell
sudo systemctl enable xfvb@:99.service --now
```

## Set up the bot systemd service and timer

Copy `luvbot-ig.service` and `luvbot-ig.timer` unit files to `/etc/systemd/system/`:

```shell
sudo cp luvbot-ig.service /etc/systemd/system/luvbot-ig.service

sudo cp luvbot-ig.timer /etc/systemd/system/luvbot-ig.timer
```

Enable the timer:

```shell
sudo systemctl enable luvbot-ig.timer --now
```