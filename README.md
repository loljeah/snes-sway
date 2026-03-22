# snes-sway

Control the Sway window manager with an SNES controller. Maps controller buttons to window management commands, app launching, and mouse control.

Built for Arduino Leonardo with [DaemonBite](https://github.com/MickGyver/DaemonBite-Retro-Controllers-USB) SNES firmware.

## Features

- **Modal input system** - Switch between navigation, input, mouse, and launcher modes
- **Mouse emulation** - D-pad controls cursor with hold-to-repeat and acceleration
- **Chord buttons** - Select/Start act as modifiers (like Shift)
- **System tray** - Mode indicator with enable/disable toggle
- **Hot-reload config** - Edit config without restarting
- **Auto-reconnect** - Daemon survives controller disconnect/reconnect
- **Mode timeout** - Auto-return to default mode after inactivity

## Installation

### Build from source

```bash
# Enter dev shell (NixOS)
nix-shell

# Build
go build -o snes-sway ./cmd/snes-sway

# Install
cp snes-sway ~/.local/bin/
```

### Dependencies

- `swaymsg` - Sway IPC (comes with Sway)
- `wtype` - Keyboard input simulation
- `wlrctl` - Mouse control (required for mouse mode)
- `notify-send` - Desktop notifications

NixOS:
```nix
environment.systemPackages = with pkgs; [ wtype wlrctl libnotify ];
```

## Usage

```bash
# Run daemon
snes-sway

# Run without system tray
snes-sway --no-tray

# Debug mode (print button events)
snes-sway --debug

# Validate config
snes-sway --validate

# Generate config interactively
snes-sway --generate-config
```

## File Locations

| File | Path | Description |
|------|------|-------------|
| Config | `~/.config/snes-sway/config.yaml` | Button mappings and settings |
| Mode file | `~/.config/snes-sway/mode` | Current mode (for waybar) |
| Systemd service | `~/.config/systemd/user/snes-sway.service` | User service unit |
| Logs | `journalctl --user -u snes-sway` | Service logs |
| Binary | `~/.local/bin/snes-sway` | Installed binary |

## Configuration

### Example config

```yaml
device:
  vendor_id: 0x2341   # Arduino
  product_id: 0x8036  # Leonardo

indicator:
  mode_file: ~/.config/snes-sway/mode
  notify: true

default_mode: navigation
mode_timeout: 30  # seconds, 0 to disable

modes:
  navigation:
    up: sway:focus up
    down: sway:focus down
    left: sway:focus left
    right: sway:focus right
    a: mode:input
    b: sway:scratchpad show
    l: sway:workspace prev
    r: sway:workspace next
    select+x: mode:mouse
    select+b: sway:kill

  input:
    up: key:Up
    down: key:Down
    left: key:Left
    right: key:Right
    a: key:Return
    b: mode:navigation

  mouse:
    up: mouse:move_up:16
    down: mouse:move_down:16
    left: mouse:move_left:16
    right: mouse:move_right:16
    a: mouse:click_left
    b: mode:navigation
    x: mouse:click_right
```

### Action types

| Type | Format | Description |
|------|--------|-------------|
| `sway` | `sway:<command>` | Run swaymsg command |
| `exec` | `exec:<command>` | Run shell command |
| `key` | `key:<keyname>` | Send keypress (wtype) |
| `mouse` | `mouse:<action>` | Mouse control (wlrctl) |
| `mode` | `mode:<name>` | Switch mode |

### Mouse actions

| Action | Description |
|--------|-------------|
| `move_up:N` | Move cursor up N pixels |
| `move_down:N` | Move cursor down N pixels |
| `move_left:N` | Move cursor left N pixels |
| `move_right:N` | Move cursor right N pixels |
| `click_left` | Left mouse click |
| `click_right` | Right mouse click |
| `click_middle` | Middle mouse click |

Mouse movement repeats while held with acceleration.

### Available buttons

**Face buttons:** `a`, `b`, `x`, `y`

**D-pad:** `up`, `down`, `left`, `right`

**Shoulders:** `l`, `r`

**Chords:** `select+<button>`, `start+<button>`

Note: `select` and `start` alone do nothing - they're modifiers only.

## Systemd Service

### Install service

```bash
mkdir -p ~/.config/systemd/user

cat > ~/.config/systemd/user/snes-sway.service << 'EOF'
[Unit]
Description=SNES Controller to Sway Window Manager
After=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/snes-sway
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now snes-sway
```

### Service commands

```bash
# Start/stop/restart
systemctl --user start snes-sway
systemctl --user stop snes-sway
systemctl --user restart snes-sway

# View logs
journalctl --user -u snes-sway -f

# Check status
systemctl --user status snes-sway
```

## Waybar Integration

Add to waybar config:

```json
"custom/snes": {
    "exec": "cat ~/.config/snes-sway/mode",
    "interval": 1,
    "format": "🎮 {}"
}
```

## Troubleshooting

### Controller not detected

```bash
# Find device
ls /dev/input/event*

# Check USB ID
lsusb | grep Arduino

# Test with evtest
sudo evtest /dev/input/event<N>
```

### Permission denied

Add udev rule:

```bash
sudo tee /etc/udev/rules.d/99-snes-controller.rules << 'EOF'
SUBSYSTEM=="input", ATTRS{idVendor}=="2341", ATTRS{idProduct}=="8036", MODE="0666"
EOF
sudo udevadm control --reload-rules
```

### Mouse mode not working

Ensure wlrctl is installed:

```bash
which wlrctl  # Should show path
wlrctl pointer move 10 10  # Test
```

## License

MIT
