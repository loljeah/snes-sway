{
  description = "SNES controller daemon for Sway window manager";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

      # Package builder — used by overlay and per-system packages
      mkSnesSway = { pkgs }: pkgs.buildGoModule {
        pname = "snes-sway";
        version = "0.1.0";

        src = ./.;

        vendorHash = "sha256-k4WEPfgkBxKfwpJ85sY7MROZhucNrSOo4BKAsjHn15o=";

        subPackages = [ "cmd/snes-sway" ];

        nativeBuildInputs = [ pkgs.makeWrapper ];

        postInstall = ''
          wrapProgram $out/bin/snes-sway \
            --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.libnotify pkgs.sway pkgs.wlrctl pkgs.wtype pkgs.dotool ]}

          # Install .desktop file for rofi/dmenu discoverability
          install -Dm644 ${./snes-sway.desktop} $out/share/applications/snes-sway.desktop

          # Install icon
          install -Dm644 ${./internal/tray/icons/navigation.png} $out/share/icons/hicolor/48x48/apps/snes-sway.png
        '';

        meta = with pkgs.lib; {
          description = "SNES controller daemon for Sway window manager";
          homepage = "https://github.com/loljeah/snes-sway";
          license = licenses.mit;
          platforms = platforms.linux;
          mainProgram = "snes-sway";
        };
      };

    in
    flake-utils.lib.eachSystem supportedSystems (system:
      let
        pkgs = import nixpkgs { inherit system; };
        snes-sway = mkSnesSway { inherit pkgs; };
      in {
        packages = {
          inherit snes-sway;
          default = snes-sway;
        };

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
            echo "  go test ./...            - run tests"
            echo "  nix build                - build package"
          '';
        };
      }
    ) // {
      # Overlay — makes pkgs.snes-sway available
      overlays.default = final: _prev: {
        snes-sway = mkSnesSway { pkgs = final; };
      };

      # NixOS system module (udev rules, input group)
      nixosModules.default = import ./nixos/module.nix;

      # Home Manager module (config, service, package)
      homeManagerModules.default = import ./nixos/hm-module.nix;
    };
}
