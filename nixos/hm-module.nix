{ config, lib, pkgs, ... }:

let
  inherit (lib) mkEnableOption mkOption mkIf types;

  cfg = config.services.snes-sway;
  yamlFormat = pkgs.formats.yaml { };

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

in {
  options.services.snes-sway = {
    enable = mkEnableOption "SNES controller daemon for Sway";

    package = lib.mkPackageOption pkgs "snes-sway" {
      default = null;
      example = "pkgs.snes-sway";
    };

    # Device options
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
        description = "USB vendor ID for device auto-detection.";
      };

      productId = mkOption {
        type = types.int;
        default = 32822; # 0x8036 — DaemonBite SNES
        description = "USB product ID for device auto-detection.";
      };
    };

    # Indicator options
    indicator = {
      modeFile = mkOption {
        type = types.str;
        default = "${config.xdg.configHome}/snes-sway/mode";
        defaultText = lib.literalExpression ''"''${config.xdg.configHome}/snes-sway/mode"'';
        description = "Path to write current mode (for waybar/status bar integration).";
      };

      notify = mkOption {
        type = types.bool;
        default = true;
        description = "Show desktop notification on mode switch.";
      };
    };

    # Mode configuration
    defaultMode = mkOption {
      type = types.str;
      default = "navigation";
      description = "Mode to activate on startup and timeout.";
    };

    modeTimeout = mkOption {
      type = types.int;
      default = 30;
      description = "Seconds of inactivity before auto-returning to default mode. Use -1 to disable.";
    };

    modes = mkOption {
      type = types.attrsOf (types.attrsOf types.str);
      default = {
        navigation = {
          up = "sway:focus up";
          down = "sway:focus down";
          left = "sway:focus left";
          right = "sway:focus right";
          a = "sway:fullscreen toggle";
          b = "sway:kill";
          x = "sway:floating toggle";
          y = "sway:layout toggle split";
          l = "sway:workspace prev";
          r = "sway:workspace next";
          "start+a" = "exec:${cfg.launcher}";
          select = "mode:launcher";
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
          "start+a" = "exec:${cfg.launcher}";
          select = "mode:navigation";
        };
      };
      description = ''
        Mode definitions mapping buttons to actions.

        Available buttons: a, b, x, y, l, r, up, down, left, right,
        select+<btn>, start+<btn>

        Action types:
        - sway:<command> — run swaymsg command
        - exec:<command> — run shell command
        - key:<keyname>  — send keypress via wtype
        - mouse:<action> — mouse control via wlrctl
        - mode:<name>    — switch to another mode
      '';
    };

    # Convenience options (used in default modes)
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

    # Freeform override — merged last, overrides everything
    settings = mkOption {
      type = yamlFormat.type;
      default = { };
      description = "Additional settings to merge into config.yaml. Overrides typed options.";
    };

    # Systemd options
    systemd = {
      enable = mkOption {
        type = types.bool;
        default = true;
        description = "Whether to create a systemd user service.";
      };

      target = mkOption {
        type = types.str;
        default = "graphical-session.target";
        description = "Systemd target to bind the service to.";
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
