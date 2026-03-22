{ config, lib, pkgs, ... }:

let
  cfg = config.services.snes-sway;

  yamlFormat = pkgs.formats.yaml { };

  defaultConfig = {
    device = {
      vendor_id = 9025;  # 0x2341
      product_id = 32822; # 0x8036
    };
    indicator = {
      mode_file = "~/.config/snes-sway/mode";
      notify = true;
    };
    default_mode = "navigation";
    modes = {
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
        start = "exec:${cfg.launcher}";
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
        start = "exec:${cfg.launcher}";
        select = "mode:navigation";
      };
    };
  };

  finalConfig = lib.recursiveUpdate defaultConfig cfg.settings;

  configFile = yamlFormat.generate "snes-sway-config.yaml" finalConfig;

  snes-sway = pkgs.buildGoModule {
    pname = "snes-sway";
    version = "0.1.0";

    src = ./..;

    vendorHash = "sha256-E5Nqlq7X5NMeHmgVWcAXzxk/MeulBI/FHy9CVxe5BCk=";

    meta = {
      description = "SNES controller daemon for Sway window manager";
      license = lib.licenses.mit;
      platforms = lib.platforms.linux;
    };
  };

in {
  options.services.snes-sway = {
    enable = lib.mkEnableOption "SNES controller daemon for Sway";

    package = lib.mkOption {
      type = lib.types.package;
      default = snes-sway;
      description = "The snes-sway package to use.";
    };

    device = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "/dev/input/event28";
      description = "Input device path. Auto-detected if null.";
    };

    launcher = lib.mkOption {
      type = lib.types.str;
      default = "wofi --show drun";
      description = "App launcher command for Start button.";
    };

    apps = {
      a = lib.mkOption {
        type = lib.types.str;
        default = "foot";
        description = "App for A button in launcher mode.";
      };
      b = lib.mkOption {
        type = lib.types.str;
        default = "firefox";
        description = "App for B button in launcher mode.";
      };
      x = lib.mkOption {
        type = lib.types.str;
        default = "thunar";
        description = "App for X button in launcher mode.";
      };
      y = lib.mkOption {
        type = lib.types.str;
        default = "code";
        description = "App for Y button in launcher mode.";
      };
    };

    settings = lib.mkOption {
      type = yamlFormat.type;
      default = { };
      description = "Additional settings to merge into config.";
    };
  };

  config = lib.mkIf cfg.enable {
    # Ensure user is in input group
    users.users.${config.users.users.ljsm.name or "ljsm"}.extraGroups = [ "input" ];

    # udev rule for consistent device access
    services.udev.extraRules = ''
      # Arduino Leonardo (DaemonBite SNES)
      SUBSYSTEM=="input", ATTRS{idVendor}=="2341", ATTRS{idProduct}=="8036", MODE="0666", TAG+="uaccess"
    '';

    # Home-manager integration (if using home-manager as NixOS module)
    # Otherwise, user should add to their home.nix
  };
}
