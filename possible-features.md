# Possible Features for snes-sway

## Current Capabilities

- **5 action types**: sway, exec, key, mouse, mode
- **34 button inputs**: 8 buttons + D-pad + Select chords + Start chords
- **Mouse control**: move, click, drag with acceleration
- **Mode system**: named modes, timeout, auto-switch
- **Integration**: systray, waybar, notifications, hot-reload

---

## Proposed Features

### A. New Action Types

#### 1. `sequence:<action1>|<action2>|<action3>` — Macro chains
Execute multiple actions in sequence with optional delays.
```yaml
a: "sequence:sway:focus left|+100ms|sway:move left"
```

#### 2. `notify:<title>::<body>` — Custom notifications
Send desktop notification without command execution.
```yaml
select+a: "notify:Mode::Switched to launcher"
```

#### 3. `script:<path>` — Script execution
Run shell script with environment variables (button, mode, etc).
```yaml
select+x: "script:~/.config/snes-sway/scripts/toggle-vpn.sh"
```

#### 4. `media:<action>` — Media controls
Control media players via MPRIS.
```yaml
l: "media:prev"
r: "media:next"
a: "media:play_pause"
```

#### 5. `dbus:<service>:<method>` — DBus integration
Direct DBus calls for system control.
```yaml
select+l: "dbus:org.bluez:PowerOff"
```

---

### B. Advanced Input Features

#### 6. Long-press Detection
Separate actions for tap vs hold (threshold: 500ms).
```yaml
a: "sway:focus right"        # tap
a+hold: "mode:launcher"      # hold
```

#### 7. Double-tap Detection
Same button pressed twice within 300ms.
```yaml
b: "sway:scratchpad show"    # single tap
b!: "sway:kill"              # double tap
```

#### 8. Multi-button Combos
Support arbitrary button combinations beyond Select/Start.
```yaml
a+b: "sway:fullscreen"
l+r: "mode:gaming"
```

#### 9. Gesture Recognition
D-pad sequence patterns trigger actions.
```yaml
gestures:
  up-up-down-down: "exec:konami-code.sh"
  circle-cw: "sway:workspace next"
```

---

### C. Advanced Mode Features

#### 10. Sticky Modes
Modes that don't timeout.
```yaml
modes:
  launcher:
    sticky: true
    b: "mode:navigation"  # explicit exit only
```

#### 11. Mode Stacking (Push/Pop)
Overlay modes that return to previous.
```yaml
a: "mode:push:help"     # push help overlay
b: "mode:pop"           # return to previous
```

#### 12. Mode Inheritance
Child modes inherit unmapped buttons from parent.
```yaml
modes:
  mouse:
    inherit: navigation
    up: "mouse:move_up:16"
    # unmapped buttons fall through to navigation
```

#### 13. Context-aware Auto-switching
Auto-switch mode based on focused window.
```yaml
contexts:
  - match: "class=firefox"
    mode: browser
  - match: "class=Steam"
    mode: gaming
```

#### 14. Per-workspace Mode Defaults
Different default mode per workspace.
```yaml
workspace_modes:
  1: navigation
  2: coding
  gaming: gaming
```

---

### D. Quality of Life

#### 15. Profile System
Multiple config profiles, switchable at runtime.
```yaml
# ~/.config/snes-sway/profiles/gaming.yaml
# Switch: snes-sway --profile gaming
```

#### 16. Macro Recording
Record and replay button sequences.
```
$ snes-sway --record my-macro
# press buttons...
$ snes-sway --stop-record

a: "macro:my-macro"
```

#### 17. Config GUI
Graphical configuration editor.
```
$ snes-sway --gui
```

#### 18. Button Analytics
Track usage patterns, suggest optimizations.
```
$ snes-sway --stats
Most used: a (1523), up (892), b (654)
Unused: select+y, start+l, start+r
```

---

### E. Accessibility

#### 19. Per-button Repeat Rates
Different repeat speeds per button.
```yaml
repeat:
  up: { delay: 100ms, interval: 16ms }
  a: { delay: 500ms, interval: 200ms }
```

#### 20. Audio Feedback
Sound cues for mode changes and actions.
```yaml
feedback:
  audio: true
  sounds:
    mode_change: ~/.config/snes-sway/sounds/mode.wav
    error: ~/.config/snes-sway/sounds/error.wav
```

#### 21. One-hand Mode
Remap for single-handed operation.
```yaml
one_hand:
  enabled: true
  mirror: left  # D-pad mirrors to ABXY
```

---

### F. Gaming & Media

#### 22. Gaming Mode
Disable timeouts, enable rapid-fire.
```yaml
modes:
  gaming:
    timeout: 0
    rapid_fire: [a, b]
```

#### 23. Game Detection
Auto-switch mode when game window focused.
```yaml
game_detection:
  enabled: true
  patterns: ["Steam", "wine", "lutris"]
  mode: gaming
```

#### 24. OBS Integration
Control OBS for streaming.
```yaml
select+r: "obs:scene:Gaming"
select+l: "obs:record:toggle"
```

---

### G. Mouse Enhancements

#### 25. Acceleration Profiles
Named curves for different use cases.
```yaml
mouse:
  profile: exponential  # linear, exponential, s-curve
  max_speed: 50
  acceleration: 0.9
```

#### 26. Scroll Mode
D-pad scrolls instead of moving cursor.
```yaml
modes:
  scroll:
    up: "mouse:scroll_up:3"
    down: "mouse:scroll_down:3"
```

#### 27. Snap-to-window
Cursor jumps to focused window center.
```yaml
select+a: "mouse:snap_to_focus"
```

---

### H. Monitoring & Debug

#### 28. Metrics Endpoint
Prometheus-style metrics.
```yaml
metrics:
  enabled: true
  port: 9100
  # GET /metrics -> button_presses_total{button="a"} 1523
```

#### 29. Event Socket
Stream events to external apps.
```yaml
socket:
  enabled: true
  path: /tmp/snes-sway.sock
  # {"button":"a","pressed":true,"mode":"navigation"}
```

#### 30. Verbose Debug Mode
Detailed logging for troubleshooting.
```
$ snes-sway --debug --verbose
[10:23:45.123] button=a pressed=true mode=navigation
[10:23:45.124] action=sway:focus right
[10:23:45.156] result=ok latency=32ms
```

---

### I. Network & Multi-device

#### 31. Remote Control
Control from phone/web.
```yaml
remote:
  enabled: true
  port: 8080
  auth: token
```

#### 32. Multi-controller Support
Handle multiple SNES controllers.
```yaml
controllers:
  - id: player1
    vendor: 0x2341
    product: 0x8036
  - id: player2
    vendor: 0x2341
    product: 0x8037
```

---

## Priority Tiers

### Tier 1 — High Impact
- Long-press detection
- Sticky modes
- Sequence actions
- Media controls
- Per-app profiles

### Tier 2 — Medium Impact
- Double-tap detection
- Mode stacking
- Macro recording
- Scroll mode
- Audio feedback

### Tier 3 — Nice to Have
- Game detection
- Config GUI
- Gesture recognition
- Multi-button combos
- Metrics endpoint

### Tier 4 — Experimental
- DBus integration
- Remote control
- Multi-controller
- Analog stick emulation

---

## Technical Notes

- New action types must not break existing configs
- Chord detection complexity increases with more patterns
- Wayland/Sway-specific features may not port to other compositors
- Performance-sensitive features (gestures, analytics) need benchmarking
