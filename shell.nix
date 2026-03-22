{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    gopls
    gotools
    go-tools
    libnotify
    # For systray (GTK3/ayatana-appindicator)
    pkg-config
    gtk3
    libayatana-appindicator
  ];

  shellHook = ''
    export GOPATH="$HOME/go"
    export PATH="$GOPATH/bin:$PATH"
    export CGO_ENABLED=1
    echo "snes-sway dev shell"
    echo "  go run ./cmd/snes-sway  - run daemon"
    echo "  go build ./cmd/snes-sway - build binary"
  '';
}
