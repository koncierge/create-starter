# Ledlight

Ledlight is a simple Linux service for controlling HUB75 LED displays, specifically setups running
Colorlight 5A-75E/5A-75B receiver cards (though it may work with others). It reads image assets from
disk, converts them into frame data, and streams those frames to one or two network interfaces using
raw Ethernet packets.

The service is intended to run on a Linux distribution using `systemd`.

## Runtime Overview

At startup, Ledlight:

1. Loads configuration from `.env` in the same directory as the executable.
2. Opens one or two raw packet sockets on the configured network interfaces.
3. Reads image files from `ASSET_PATH`.
4. Resizes and crops images to the configured display dimensions.
5. Streams each image to the display for `SLIDE_DURATION` seconds.
6. Watches for asset list changes between playback loops.

Supported image formats:

-   PNG
-   JPEG
-   GIF

## Requirements

Server requirements:

-   Debian-based Linux distribution
-   Go 1.18 or newer, if building on the server
-   `setcap` and `getcap`, if running without root
-   One or two physical network interfaces connected to the display receivers
-   Image assets available on local disk or a mounted filesystem
-   A valid `.env` file next to the binary

The service uses Linux-only raw packet sockets, so it will not run natively on macOS or Windows.

## Expected File Layout

The default service configuration expects this layout:

```text
/opt/ledlight/
├── run
├── .env
└── fallback.png
```

`fallback.png` is the emergency image used when the live asset path cannot be read, or when no
selected image can be decoded.

Logs are written to:

```text
/var/log/ledlight/YYYY-MM-DD.log
```

## Environment Configuration

Ledlight loads `/opt/ledlight/.env` because the executable runs from `/opt/ledlight/run`. The `.env`
file is resolved from the executable directory, not from the current working directory.

### Environment Variables

| Variable           | Description                                                               |
| ------------------ | ------------------------------------------------------------------------- |
| `DEVICE_OUTPUT`    | `single` or `dual`. Enables one or two display outputs.                   |
| `DEVICE_IFACE`     | Primary Linux network interface name.                                     |
| `DEVICE_AUX_IFACE` | Auxiliary Linux network interface name, used when `DEVICE_OUTPUT="dual"`. |
| `DEVICE_IFACE_PWM` | Uses the PWM source MAC address for the primary interface.                |
| `DEVICE_AUX_PWM`   | Uses the PWM source MAC address for the auxiliary interface.              |
| `LOCALE`           | IANA timezone used for day/night brightness scheduling.                   |
| `ASSET_PATH`       | Directory containing live display images.                                 |
| `DISPLAY_WIDTH`    | Display width in pixels.                                                  |
| `DISPLAY_HEIGHT`   | Display height in pixels.                                                 |
| `FRAME_RATE`       | Target frame rate.                                                        |
| `DAY_START`        | Start of daytime brightness window, in `HH:MM` format.                    |
| `DAY_END`          | End of daytime brightness window, in `HH:MM` format.                      |
| `BRIGHTNESS_DAY`   | Brightness percentage used during the daytime window.                     |
| `BRIGHTNESS_NIGHT` | Brightness percentage used outside the daytime window.                    |
| `SUBMASK_*`        | Auxiliary output crop settings, used for dual output.                     |
| `DEFAULT_SLIDE`    | Image used when `ASSET_PATH` is readable but contains no live assets.     |
| `SLIDE_DURATION`   | Seconds to display each slide.                                            |

Playback image selection is:

1. Live images from `ASSET_PATH`, when the directory is readable and contains image files.
2. `DEFAULT_SLIDE`, when `ASSET_PATH` is readable but empty.
3. `/opt/ledlight/fallback.png`, when `ASSET_PATH` cannot be read or the selected images cannot be
   decoded.

## Setup Guide

These steps assume you have access to this repository on the target Linux server.

Run the setup script from the repository root:

```bash
sudo ./setup.sh --start
```

For dual output:

```bash
sudo ./setup.sh --output dual --start
```

The script does the following:

1. Detects display network interfaces.
2. Installs missing dependencies.
3. Creates the required directories, installs binary and artifacts.
4. Creates the `.env` file.
5. Creates the service user and log directory.
6. Installs and starts the systemd service.
7. Creates asset folders in the home directory and sets permissions.

Some of these behaviours are configurable with script flags. Run `./setup.sh --help` for details.

For example, to use an existing environment file or default slide:

```bash
sudo ./setup.sh --env my.env --default-slide /path/to/asset/root/default.png --start
```

If creating `/opt/ledlight/.env` manually, use absolute paths.

### Check Configuration

Review the generated environment file:

```bash
sudo nano /opt/ledlight/.env
```

Confirm the deployment-specific values:

-   `DEVICE_OUTPUT` is `single` or `dual`.
-   `DEVICE_IFACE` is the NIC connected to the primary display controller.
-   `DEVICE_AUX_IFACE` is the second controller NIC when using dual output.
-   `DEVICE_IFACE_PWM` and `DEVICE_AUX_PWM` match the receiver card type.
-   `DISPLAY_WIDTH` and `DISPLAY_HEIGHT` match the panel layout.
-   `ASSET_PATH` and `DEFAULT_SLIDE` are absolute paths and readable by the `ledlight` user.

Check available interfaces and compare them with the `.env` values:

```bash
ip -br link
```

Check the binary capability:

```bash
getcap /opt/ledlight/run
```

Expected output:

```text
/opt/ledlight/run cap_net_raw=ep
```

Check service status:

```bash
sudo systemctl status ledlight
```

### Assets

Add live display images to:

```bash
~/ledlight-assets/live
```

After changing asset files, ensure the service user can read them:

```bash
sudo chown -R ledlight:ledlight ~/ledlight-assets
```

### Logs

Application logs:

```bash
sudo tail -f /var/log/ledlight/$(date +%F).log
```

systemd logs:

```bash
sudo journalctl -u ledlight -f
```

### Restart After Changes

Restart the service after editing `/opt/ledlight/.env`, replacing `/opt/ledlight/run`, changing the
systemd service file, or changing interface cabling:

```bash
sudo systemctl restart ledlight
```

If the systemd service file changes, reload systemd first:

```bash
sudo systemctl daemon-reload
sudo systemctl restart ledlight
```

## Updating the Binary

After pulling new code or replacing files in this repository, rerun setup from the repository root:

```bash
sudo ./setup.sh --start
```

The script rebuilds `dist/run` when needed, installs it to `/opt/ledlight/run`, reapplies
`CAP_NET_RAW`, and restarts the service when `--start` is used. Check the service and application
logs after restart.
