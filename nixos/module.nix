{ config, lib, pkgs, ... }:

let
  inherit (lib) mkEnableOption mkOption mkIf types;

  cfg = config.services.snes-sway;

in {
  options.services.snes-sway = {
    enable = mkEnableOption "SNES controller system support (udev rules, input group)";

    user = mkOption {
      type = types.str;
      description = "User to add to the input group for device access.";
      example = "ljsm";
    };

    device = {
      vendorId = mkOption {
        type = types.str;
        default = "2341";
        description = "USB vendor ID (hex, no prefix) for udev rule.";
        example = "2341";
      };

      productId = mkOption {
        type = types.str;
        default = "8036";
        description = "USB product ID (hex, no prefix) for udev rule.";
        example = "8036";
      };
    };
  };

  config = mkIf cfg.enable {
    # Add user to input group for evdev access
    users.users.${cfg.user}.extraGroups = [ "input" ];

    # udev rule for consistent device permissions
    services.udev.extraRules = ''
      # SNES controller (DaemonBite / Arduino Leonardo)
      SUBSYSTEM=="input", ATTRS{idVendor}=="${cfg.device.vendorId}", ATTRS{idProduct}=="${cfg.device.productId}", MODE="0660", GROUP="input", TAG+="uaccess"
    '';
  };
}
