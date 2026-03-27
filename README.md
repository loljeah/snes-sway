# snes-sway

Control the Sway window manager with an SNES controller. Maps controller buttons to window management commands, app launching, and mouse control.

Built for Arduino Leonardo with [DaemonBite](https://github.com/MickGyver/DaemonBite-Retro-Controllers-USB) SNES firmware.

## Features

- **Modal input system** - Switch between navigation, input, mouse, launcher, and drag modes
- **Mouse emulation** - D-pad controls cursor with hold-to-repeat and acceleration
- **Chord buttons** - Select/Start act as modifiers (like Shift)
- **Hot-reload config** - Edit config without restarting
- **Auto-reconnect** - Daemon survives controller disconnect/reconnect
- **Mode timeout** - Auto-return to default mode after inactivity
- **Headless operation** - No system tray, managed via systemd

## Installation

### NixOS with Home Manager (Recommended)

Add the flake to your inputs:

```nix
# flake.nix
{
  inputs = {
    snes-sway.url = "github:ljsm/snes-sway";
  };
}
```

Use the overlay and modules:

```nix
# configuration.nix (system-level for udev rules)
{ inputs, ... }:
{
  imports = [ inputs.snes-sway.nixosModules.default ];

  nixpkgs.overlays = [ inputs.snes-sway.overlays.default ];

  services.snes-sway = {
    enable = true;
    user = "your-username";
  };
}
```

```nix
# home.nix (Home Manager for user config)
{ inputs, pkgs, ... }:
{
  imports = [ inputs.snes-sway.homeManagerModules.default ];

  services.snes-sway = {
    enable = true;
    package = pkgs.snes-sway;

    # Customize launcher
    launcher = "wofi --show drun";

    # Customize quick-launch apps
    apps = {
      a = "foot";
      b = "firefox";
      x = "thunar";
      y = "code";
    };

    # Mouse speed settings
    mouse = {
      speed = 16;           # Normal speed (pixels)
      fastSpeed = 50;       # Shoulder buttons (pixels)
      precisionSpeed = 4;   # Select+D-pad (pixels)
    };

    # Mode timeout (seconds, -1 to disable)
    modeTimeout = 30;

    # Enable waybar integration
    waybar.enable = true;
  };
}
```

### Build from source

```bash
# Enter dev shell (NixOS)
nix develop

# Build
go build -o snes-sway ./cmd/snes-sway

# Install
cp snes-sway ~/.local/bin/
```

### Dependencies

- `swaymsg` - Sway IPC (comes with Sway)
- `wtype` - Keyboard input simulation
- `wlrctl` - Mouse control (required for mouse mode)
- `dotool` - Drag/select operations
- `notify-send` - Desktop notifications

NixOS:
```nix
environment.systemPackages = with pkgs; [ wtype wlrctl dotool libnotify ];
```

## Usage

```bash
# Run daemon
snes-sway

# Debug mode (print button events)
snes-sway --debug

# Validate config
snes-sway --validate

# Generate config interactively
snes-sway --generate-config

# Custom config path
snes-sway --config /path/to/config.yaml
```

## Home Manager Configuration

### Full Options Reference

```nix
services.snes-sway = {
  enable = true;
  package = pkgs.snes-sway;

  # ── Device Detection ──────────────────────────────────────
  device = {
    path = null;                    # Auto-detect (or "/dev/input/eventN")
    vendorId = 9025;                # 0x2341 (Arduino Leonardo)
    productId = 32822;              # 0x8036 (DaemonBite SNES)
  };

  # ── Mode Indicator ────────────────────────────────────────
  indicator = {
    modeFile = "~/.config/snes-sway/mode";   # For waybar
    notify = true;                           # Desktop notifications
  };

  # ── Mode Settings ─────────────────────────────────────────
  defaultMode = "navigation";       # Startup mode
  modeTimeout = 30;                 # Auto-return to default (seconds)

  # ── Quick Launch ──────────────────────────────────────────
  launcher = "rofi -show drun";     # Start+A
  apps = {
    a = "foot";                     # Launcher mode: A
    b = "firefox";                  # Launcher mode: B
    x = "thunar";                   # Launcher mode: X
    y = "code";                     # Launcher mode: Y
  };

  # ── Mouse Settings ────────────────────────────────────────
  mouse = {
    speed = 16;                     # D-pad movement (pixels)
    fastSpeed = 50;                 # L/R shoulders (pixels)
    precisionSpeed = 4;             # Select+D-pad (pixels)
  };

  # ── Systemd Service ───────────────────────────────────────
  systemd = {
    enable = true;
    target = "sway-session.target";
  };

  # ── Waybar Integration ────────────────────────────────────
  waybar = {
    enable = true;
    moduleName = "custom/snes-mode";
    format = "{icon} {}";
    formatIcons = {
      navigation = "";
      launcher = "";
      input = "";
      mouse = "";
      drag = "";
      default = "";
    };
  };

  # ── Custom Mode Definitions ───────────────────────────────
  modes = {
    navigation = {
      up = "sway:focus up";
      down = "sway:focus down";
      # ... full mode configuration
    };
    # Add custom modes here
  };

  # ── Freeform Override (merged last) ───────────────────────
  settings = {
    # Any raw YAML config options
  };
};
```

### Custom Mode Example

```nix
services.snes-sway.modes = {
  # Override navigation mode
  navigation = {
    up = "sway:focus up";
    down = "sway:focus down";
    left = "sway:focus left";
    right = "sway:focus right";
    a = "mode:input";
    b = "sway:scratchpad show";
    x = "sway:floating toggle";
    y = "sway:layout toggle split";
    l = "sway:workspace prev";
    r = "sway:workspace next";
    "select+a" = "sway:fullscreen toggle";
    "select+b" = "sway:kill";
    "select+x" = "mode:mouse";
    "start+a" = "exec:wofi --show drun";
    "start+b" = "mode:launcher";
  };

  # Add a custom media mode
  media = {
    up = "exec:wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%+";
    down = "exec:wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-";
    a = "exec:playerctl play-pause";
    b = "mode:navigation";
    l = "exec:playerctl previous";
    r = "exec:playerctl next";
    "select+a" = "exec:wpctl set-mute @DEFAULT_AUDIO_SINK@ toggle";
  };
};
```

## File Locations

| File | Path | Description |
|------|------|-------------|
| Config | `~/.config/snes-sway/config.yaml` | Button mappings and settings |
| Mode file | `~/.config/snes-sway/mode` | Current mode (for waybar) |
| Waybar snippet | `~/.config/snes-sway/waybar-module.json` | Waybar module config |
| Logs | `journalctl --user -u snes-sway` | Service logs |

## Configuration

### Action Types

| Type | Format | Description |
|------|--------|-------------|
| `sway` | `sway:<command>` | Run swaymsg command |
| `exec` | `exec:<command>` | Run shell command |
| `key` | `key:<keyname>` | Send keypress (wtype) |
| `mouse` | `mouse:<action>` | Mouse control (wlrctl) |
| `mode` | `mode:<name>` | Switch mode |

### Mouse Actions

| Action | Description |
|--------|-------------|
| `move_up:N` | Move cursor up N pixels |
| `move_down:N` | Move cursor down N pixels |
| `move_left:N` | Move cursor left N pixels |
| `move_right:N` | Move cursor right N pixels |
| `click_left` | Left mouse click |
| `click_right` | Right mouse click |
| `click_middle` | Middle mouse click |
| `double_left` | Double-click (word select) |
| `hold_left` | Hold left button (drag start) |
| `release_left` | Release left button (drag end) |

Mouse movement repeats while held with acceleration.

### Available Buttons

**Face buttons:** `a`, `b`, `x`, `y`

**D-pad:** `up`, `down`, `left`, `right`

**Shoulders:** `l`, `r`

**Chords:** `select+<button>`, `start+<button>`

Note: `select` and `start` alone do nothing - they're modifiers only.

## Default Mode Bindings

### Navigation Mode (default)
| Button | Action |
|--------|--------|
| D-pad | Focus window |
| A | Enter input mode |
| B | Toggle scratchpad |
| X | Toggle floating |
| Y | Toggle layout |
| L/R | Prev/next workspace |
| Select+A | Toggle fullscreen |
| Select+B | Kill window |
| Select+X | Enter mouse mode |
| Start+A | Open launcher |
| Start+B | Enter launcher mode |

### Mouse Mode
| Button | Action |
|--------|--------|
| D-pad | Move cursor |
| A | Left click |
| B | Return to navigation |
| X | Right click |
| Y | Middle click |
| L/R | Fast horizontal move |
| Select+D-pad | Precision movement |
| Select+A | Double-click |
| Select+X | Enter drag mode |

### Input Mode
| Button | Action |
|--------|--------|
| D-pad | Arrow keys |
| A | Enter/Return |
| B | Return to navigation |
| X | Escape |
| Y | Tab |
| L/R | Page Up/Down |

## Systemd Service

### Commands

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

If using Home Manager with `waybar.enable = true`, a module config is generated at `~/.config/snes-sway/waybar-module.json`.

Add to your waybar config:

```json
"custom/snes-mode": {
    "exec": "cat ~/.config/snes-sway/mode 2>/dev/null || echo 'off'",
    "interval": 1,
    "format": "{} "
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

NixOS system module creates udev rules automatically. For manual setup:

```bash
sudo tee /etc/udev/rules.d/99-snes-controller.rules << 'EOF'
SUBSYSTEM=="input", ATTRS{idVendor}=="2341", ATTRS{idProduct}=="8036", MODE="0660", GROUP="input", TAG+="uaccess"
EOF
sudo udevadm control --reload-rules
```

### Mouse mode not working

Ensure wlrctl is installed:

```bash
which wlrctl  # Should show path
wlrctl pointer move 10 10  # Test
```

### Drag mode not working

Ensure dotool is installed and has permission to access `/dev/uinput`:

```bash
which dotool
echo "buttondown left" | dotool  # Test
```

## Security

The daemon implements several security measures:

- **Config ownership validation** - Config must be owned by current user, not world-writable
- **NixOS store symlinks allowed** - Home Manager managed configs via `/nix/store/` are trusted
- **Command whitelist** - Only known-safe commands are passed to dotool
- **Path traversal prevention** - Mode file path validated to be under home directory
- **Command timeout** - All external commands have 5-second timeout
- **Systemd hardening** - Service runs with strict sandboxing

## License

MIT
