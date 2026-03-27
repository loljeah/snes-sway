{ config, lib, pkgs, ... }:

let
  inherit (lib) mkEnableOption mkOption mkIf types literalExpression;

  cfg = config.services.snes-sway;
  yamlFormat = pkgs.formats.yaml { };

  # All available SNES buttons
  buttonNames = [
    "a" "b" "x" "y"
    "l" "r"
    "up" "down" "left" "right"
    "select+a" "select+b" "select+x" "select+y"
    "select+l" "select+r"
    "select+up" "select+down" "select+left" "select+right"
    "start+a" "start+b" "start+x" "start+y"
    "start+l" "start+r"
    "start+up" "start+down" "start+left" "start+right"
  ];

  # Action type documentation
  actionDoc = ''
    Action format: type:command

    Types:
    - sway:<command>    Run swaymsg command (focus, workspace, fullscreen, etc.)
    - exec:<command>    Run shell command via swaymsg exec
    - key:<keyname>     Send keypress via wtype (Up, Down, Return, Escape, etc.)
    - mouse:<action>    Mouse control via wlrctl/dotool
    - mode:<name>       Switch to another mode

    Mouse actions:
    - click_left, click_right, click_middle
    - double_left (word select)
    - hold_left, hold_right, release_left, release_right (drag)
    - move_up:N, move_down:N, move_left:N, move_right:N (pixels)
  '';

  # Build the YAML config from typed options, then merge freeform settings
  generatedConfig = {
    device = {
      vendor_id = cfg.device.vendorId;
      product_id = cfg.device.productId;
    } // lib.optionalAttrs (cfg.device.path != null) {
      path = cfg.device.path;
    };

    indicator = {
      mode_file = cfg.indicator.modeFile;
      notify = cfg.indicator.notify;
    };

    default_mode = cfg.defaultMode;
    mode_timeout = cfg.modeTimeout;

    modes = cfg.modes;
  };

  finalConfig = lib.recursiveUpdate generatedConfig cfg.settings;

  # Mode option type with all buttons
  modeType = types.attrsOf types.str;

  # Default mode configurations
  defaultModes = {
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
      "select+l" = "sway:focus output left";
      "select+r" = "sway:focus output right";
      "start+a" = "exec:${cfg.launcher}";
      "start+b" = "mode:launcher";
    };

    launcher = {
      up = "sway:focus up";
      down = "sway:focus down";
      left = "sway:focus left";
      right = "sway:focus right";
      a = "exec:${cfg.apps.a}";
      b = "exec:${cfg.apps.b}";
      x = "exec:${cfg.apps.x}";
      y = "exec:${cfg.apps.y}";
      l = "sway:workspace prev";
      r = "sway:workspace next";
      "select+b" = "sway:kill";
      "start+a" = "exec:${cfg.launcher}";
      "start+b" = "mode:navigation";
    };

    input = {
      up = "key:Up";
      down = "key:Down";
      left = "key:Left";
      right = "key:Right";
      a = "key:Return";
      b = "mode:navigation";
      x = "key:Escape";
      y = "key:Tab";
      l = "key:Page_Up";
      r = "key:Page_Down";
      "select+b" = "sway:kill";
      "select+l" = "sway:focus output left";
      "select+r" = "sway:focus output right";
    };

    mouse = {
      up = "mouse:move_up:${toString cfg.mouse.speed}";
      down = "mouse:move_down:${toString cfg.mouse.speed}";
      left = "mouse:move_left:${toString cfg.mouse.speed}";
      right = "mouse:move_right:${toString cfg.mouse.speed}";
      a = "mouse:click_left";
      b = "mode:navigation";
      x = "mouse:click_right";
      y = "mouse:click_middle";
      l = "mouse:move_left:${toString cfg.mouse.fastSpeed}";
      r = "mouse:move_right:${toString cfg.mouse.fastSpeed}";
      "select+up" = "mouse:move_up:${toString cfg.mouse.precisionSpeed}";
      "select+down" = "mouse:move_down:${toString cfg.mouse.precisionSpeed}";
      "select+left" = "mouse:move_left:${toString cfg.mouse.precisionSpeed}";
      "select+right" = "mouse:move_right:${toString cfg.mouse.precisionSpeed}";
      "select+a" = "mouse:double_left";
      "select+x" = "mode:drag";
      "select+b" = "sway:kill";
    };

    drag = {
      up = "mouse:move_up:${toString cfg.mouse.speed}";
      down = "mouse:move_down:${toString cfg.mouse.speed}";
      left = "mouse:move_left:${toString cfg.mouse.speed}";
      right = "mouse:move_right:${toString cfg.mouse.speed}";
      a = "mouse:release_left";
      b = "mouse:release_left";
      l = "mouse:move_left:${toString cfg.mouse.fastSpeed}";
      r = "mouse:move_right:${toString cfg.mouse.fastSpeed}";
      "select+up" = "mouse:move_up:${toString cfg.mouse.precisionSpeed}";
      "select+down" = "mouse:move_down:${toString cfg.mouse.precisionSpeed}";
      "select+left" = "mouse:move_left:${toString cfg.mouse.precisionSpeed}";
      "select+right" = "mouse:move_right:${toString cfg.mouse.precisionSpeed}";
    };
  };

in {
  options.services.snes-sway = {
    enable = mkEnableOption "SNES controller daemon for Sway";

    package = lib.mkPackageOption pkgs "snes-sway" {
      default = null;
      example = "pkgs.snes-sway";
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Device Configuration
    # ─────────────────────────────────────────────────────────────────────────

    device = {
      path = mkOption {
        type = types.nullOr types.str;
        default = null;
        example = "/dev/input/event28";
        description = "Input device path. Auto-detected from vendor/product ID if null.";
      };

      vendorId = mkOption {
        type = types.int;
        default = 9025; # 0x2341 — Arduino Leonardo
        description = "USB vendor ID for device auto-detection (decimal).";
      };

      productId = mkOption {
        type = types.int;
        default = 32822; # 0x8036 — DaemonBite SNES
        description = "USB product ID for device auto-detection (decimal).";
      };
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Indicator Configuration
    # ─────────────────────────────────────────────────────────────────────────

    indicator = {
      modeFile = mkOption {
        type = types.str;
        default = "${config.xdg.configHome}/snes-sway/mode";
        defaultText = literalExpression ''"''${config.xdg.configHome}/snes-sway/mode"'';
        description = "Path to write current mode (for waybar/status bar integration).";
      };

      notify = mkOption {
        type = types.bool;
        default = true;
        description = "Show desktop notification on mode switch.";
      };
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Mode Configuration
    # ─────────────────────────────────────────────────────────────────────────

    defaultMode = mkOption {
      type = types.str;
      default = "navigation";
      description = "Mode to activate on startup and after timeout.";
    };

    modeTimeout = mkOption {
      type = types.int;
      default = 30;
      description = "Seconds of inactivity before auto-returning to default mode. Use -1 to disable.";
    };

    modes = mkOption {
      type = types.attrsOf modeType;
      default = defaultModes;
      description = ''
        Mode definitions mapping buttons to actions.

        Available buttons: ${lib.concatStringsSep ", " buttonNames}

        ${actionDoc}
      '';
      example = literalExpression ''
        {
          navigation = {
            up = "sway:focus up";
            down = "sway:focus down";
            a = "mode:input";
            "select+b" = "sway:kill";
          };
          custom = {
            a = "exec:firefox";
            b = "mode:navigation";
          };
        }
      '';
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Convenience Options — Apps & Launcher
    # ─────────────────────────────────────────────────────────────────────────

    launcher = mkOption {
      type = types.str;
      default = "rofi -show drun";
      description = "App launcher command for Start+A button.";
    };

    apps = {
      a = mkOption {
        type = types.str;
        default = "foot";
        description = "App for A button in launcher mode.";
      };
      b = mkOption {
        type = types.str;
        default = "firefox";
        description = "App for B button in launcher mode.";
      };
      x = mkOption {
        type = types.str;
        default = "thunar";
        description = "App for X button in launcher mode.";
      };
      y = mkOption {
        type = types.str;
        default = "code";
        description = "App for Y button in launcher mode.";
      };
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Mouse Configuration
    # ─────────────────────────────────────────────────────────────────────────

    mouse = {
      speed = mkOption {
        type = types.int;
        default = 16;
        description = "Default mouse movement speed in pixels.";
      };

      fastSpeed = mkOption {
        type = types.int;
        default = 50;
        description = "Fast mouse movement speed (shoulders) in pixels.";
      };

      precisionSpeed = mkOption {
        type = types.int;
        default = 4;
        description = "Precision mouse movement speed (select+dpad) in pixels.";
      };
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Navigation Mode Shortcuts
    # ─────────────────────────────────────────────────────────────────────────

    navigation = {
      focus = {
        up = mkOption {
          type = types.str;
          default = "sway:focus up";
          description = "Action for D-pad up in navigation mode.";
        };
        down = mkOption {
          type = types.str;
          default = "sway:focus down";
          description = "Action for D-pad down in navigation mode.";
        };
        left = mkOption {
          type = types.str;
          default = "sway:focus left";
          description = "Action for D-pad left in navigation mode.";
        };
        right = mkOption {
          type = types.str;
          default = "sway:focus right";
          description = "Action for D-pad right in navigation mode.";
        };
      };

      window = {
        fullscreen = mkOption {
          type = types.str;
          default = "sway:fullscreen toggle";
          description = "Fullscreen toggle action.";
        };
        floating = mkOption {
          type = types.str;
          default = "sway:floating toggle";
          description = "Floating toggle action.";
        };
        kill = mkOption {
          type = types.str;
          default = "sway:kill";
          description = "Kill window action.";
        };
        layout = mkOption {
          type = types.str;
          default = "sway:layout toggle split";
          description = "Layout toggle action.";
        };
        scratchpad = mkOption {
          type = types.str;
          default = "sway:scratchpad show";
          description = "Scratchpad toggle action.";
        };
      };

      workspace = {
        prev = mkOption {
          type = types.str;
          default = "sway:workspace prev";
          description = "Previous workspace action.";
        };
        next = mkOption {
          type = types.str;
          default = "sway:workspace next";
          description = "Next workspace action.";
        };
        outputLeft = mkOption {
          type = types.str;
          default = "sway:focus output left";
          description = "Focus left output action.";
        };
        outputRight = mkOption {
          type = types.str;
          default = "sway:focus output right";
          description = "Focus right output action.";
        };
      };
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Input Mode Shortcuts
    # ─────────────────────────────────────────────────────────────────────────

    input = {
      keys = {
        up = mkOption {
          type = types.str;
          default = "key:Up";
          description = "Key to send for D-pad up in input mode.";
        };
        down = mkOption {
          type = types.str;
          default = "key:Down";
          description = "Key to send for D-pad down in input mode.";
        };
        left = mkOption {
          type = types.str;
          default = "key:Left";
          description = "Key to send for D-pad left in input mode.";
        };
        right = mkOption {
          type = types.str;
          default = "key:Right";
          description = "Key to send for D-pad right in input mode.";
        };
        confirm = mkOption {
          type = types.str;
          default = "key:Return";
          description = "Confirm key (A button).";
        };
        cancel = mkOption {
          type = types.str;
          default = "key:Escape";
          description = "Cancel key (X button).";
        };
        tab = mkOption {
          type = types.str;
          default = "key:Tab";
          description = "Tab key (Y button).";
        };
        pageUp = mkOption {
          type = types.str;
          default = "key:Page_Up";
          description = "Page up key (L button).";
        };
        pageDown = mkOption {
          type = types.str;
          default = "key:Page_Down";
          description = "Page down key (R button).";
        };
      };
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Freeform Override
    # ─────────────────────────────────────────────────────────────────────────

    settings = mkOption {
      type = yamlFormat.type;
      default = { };
      description = "Additional settings to merge into config.yaml. Overrides typed options.";
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Systemd Options
    # ─────────────────────────────────────────────────────────────────────────

    systemd = {
      enable = mkOption {
        type = types.bool;
        default = true;
        description = "Whether to create a systemd user service.";
      };

      target = mkOption {
        type = types.str;
        default = "sway-session.target";
        description = "Systemd target to bind the service to.";
      };
    };

    # ─────────────────────────────────────────────────────────────────────────
    # Waybar Integration
    # ─────────────────────────────────────────────────────────────────────────

    waybar = {
      enable = mkOption {
        type = types.bool;
        default = false;
        description = "Whether to install waybar module configuration snippet.";
      };

      moduleName = mkOption {
        type = types.str;
        default = "custom/snes-mode";
        description = "Waybar module name for SNES mode display.";
      };

      format = mkOption {
        type = types.str;
        default = "{icon} {}";
        description = "Waybar format string for mode display.";
      };

      formatIcons = mkOption {
        type = types.attrsOf types.str;
        default = {
          navigation = "";
          launcher = "";
          input = "";
          mouse = "";
          drag = "";
          default = "";
        };
        description = "Icons for each mode in waybar.";
      };
    };
  };

  config = mkIf cfg.enable {
    assertions = [
      {
        assertion = cfg.package != null;
        message = "services.snes-sway.package must be set. Use the flake overlay or pass the package directly.";
      }
    ];

    home.packages = [ cfg.package ];

    xdg.configFile."snes-sway/config.yaml".source =
      yamlFormat.generate "snes-sway-config.yaml" finalConfig;

    # Waybar module snippet
    xdg.configFile."snes-sway/waybar-module.json" = mkIf cfg.waybar.enable {
      text = builtins.toJSON {
        "${cfg.waybar.moduleName}" = {
          exec = "cat ${cfg.indicator.modeFile} 2>/dev/null || echo 'off'";
          interval = 1;
          format = cfg.waybar.format;
          format-icons = cfg.waybar.formatIcons;
          tooltip = "SNES controller mode";
        };
      };
    };

    systemd.user.services.snes-sway = mkIf cfg.systemd.enable {
      Unit = {
        Description = "SNES Controller to Sway Window Manager";
        Documentation = [ "https://github.com/ljsm/snes-sway" ];
        After = [ cfg.systemd.target ];
        PartOf = [ cfg.systemd.target ];
        ConditionEnvironment = "WAYLAND_DISPLAY";
      };

      Service = {
        Type = "simple";
        ExecStart = "${cfg.package}/bin/snes-sway";
        Restart = "on-failure";
        RestartSec = 3;

        # Hardening
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = "read-only";
        ReadWritePaths = [ "%h/.config/snes-sway" ];
        PrivateTmp = true;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectKernelLogs = true;
        ProtectControlGroups = true;
        ProtectHostname = true;
        ProtectClock = true;
        RestrictNamespaces = true;
        RestrictRealtime = true;
        MemoryDenyWriteExecute = true;
        LockPersonality = true;
        RestrictAddressFamilies = [ "AF_UNIX" "AF_NETLINK" ];
        SystemCallFilter = [ "@system-service" "~@privileged" ];
        CapabilityBoundingSet = "";
      };

      Install = {
        WantedBy = [ cfg.systemd.target ];
      };
    };
  };
}
