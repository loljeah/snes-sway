{
  description = "SNES controller daemon for Sway window manager";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        snes-sway = pkgs.buildGoModule {
          pname = "snes-sway";
          version = "0.1.0";

          src = ./.;

          vendorHash = "sha256-k4WEPfgkBxKfwpJ85sY7MROZhucNrSOo4BKAsjHn15o=";

          subPackages = [ "cmd/snes-sway" ];

          # Runtime dependencies for notifications
          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            wrapProgram $out/bin/snes-sway \
              --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.libnotify pkgs.sway ]}
          '';

          meta = with pkgs.lib; {
            description = "SNES controller daemon for Sway window manager";
            homepage = "https://github.com/loljeah/snes-sway";
            license = licenses.mit;
            platforms = platforms.linux;
            mainProgram = "snes-sway";
          };
        };

      in {
        packages = {
          inherit snes-sway;
          default = snes-sway;
        };

        # Development shell
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            libnotify
            sway
          ];

          shellHook = ''
            echo "snes-sway dev shell"
            echo "  go run ./cmd/snes-sway   - run daemon"
            echo "  go build ./cmd/snes-sway - build binary"
            echo "  nix build                - build flake package"
          '';
        };
      }
    ) // {
      # Home Manager module
      homeManagerModules.default = { config, lib, pkgs, ... }:
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
        in {
          options.services.snes-sway = {
            enable = lib.mkEnableOption "SNES controller daemon for Sway";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.default;
              description = "The snes-sway package to use.";
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
            home.packages = [ cfg.package ];

            xdg.configFile."snes-sway/config.yaml".source =
              yamlFormat.generate "snes-sway-config.yaml" finalConfig;

            systemd.user.services.snes-sway = {
              Unit = {
                Description = "SNES Controller to Sway Window Manager";
                Documentation = "https://github.com/loljeah/snes-sway";
                After = [ "graphical-session.target" ];
                PartOf = [ "graphical-session.target" ];
              };

              Service = {
                Type = "simple";
                ExecStart = "${cfg.package}/bin/snes-sway";
                Restart = "on-failure";
                RestartSec = 3;
              };

              Install = {
                WantedBy = [ "graphical-session.target" ];
              };
            };
          };
        };
    };
}
