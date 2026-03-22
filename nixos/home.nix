{ config, lib, pkgs, ... }:

let
  cfg = config.services.snes-sway;

  yamlFormat = pkgs.formats.yaml { };

  defaultConfig = {
    device = {
      vendor_id = 9025;
      product_id = 32822;
    };
    indicator = {
      mode_file = "${config.xdg.configHome}/snes-sway/mode";
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

  snes-sway = pkgs.buildGoModule {
    pname = "snes-sway";
    version = "0.1.0";
    src = cfg.src;
    vendorHash = cfg.vendorHash;
    meta.description = "SNES controller daemon for Sway";
  };

in {
  options.services.snes-sway = {
    enable = lib.mkEnableOption "SNES controller daemon for Sway";

    src = lib.mkOption {
      type = lib.types.path;
      description = "Path to snes-sway source.";
    };

    vendorHash = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = "sha256-E5Nqlq7X5NMeHmgVWcAXzxk/MeulBI/FHy9CVxe5BCk=";
      description = "Vendor hash for Go dependencies.";
    };

    launcher = lib.mkOption {
      type = lib.types.str;
      default = "wofi --show drun";
    };

    apps = {
      a = lib.mkOption { type = lib.types.str; default = "foot"; };
      b = lib.mkOption { type = lib.types.str; default = "firefox"; };
      x = lib.mkOption { type = lib.types.str; default = "thunar"; };
      y = lib.mkOption { type = lib.types.str; default = "code"; };
    };

    settings = lib.mkOption {
      type = yamlFormat.type;
      default = { };
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ snes-sway ];

    xdg.configFile."snes-sway/config.yaml".source =
      yamlFormat.generate "snes-sway-config.yaml" finalConfig;

    systemd.user.services.snes-sway = {
      Unit = {
        Description = "SNES Controller to Sway Window Manager";
        After = [ "graphical-session.target" ];
        PartOf = [ "graphical-session.target" ];
        ConditionEnvironment = "WAYLAND_DISPLAY";
      };

      Service = {
        Type = "simple";
        ExecStart = "${snes-sway}/bin/snes-sway";
        Restart = "on-failure";
        RestartSec = 3;
      };

      Install = {
        WantedBy = [ "sway-session.target" ];
      };
    };
  };
}
