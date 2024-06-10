{ pkgs ? import <nixpkgs> { } }:

pkgs.mkShell {
  packages = with pkgs; [
    go
    templ
    sqlite
    wgo
    (writeShellScriptBin "run" ''
      go build .
      sudo setcap 'cap_net_bind_service=+ep' sods
      ./sods
    '')
  ];

  buildInputs = with pkgs; [
    go
    templ
  ];

  buildPhase = ''
    go mod tidy
    templ generate
    sqlite3 sodsdb.sqlite ".read sodsSchema.sql"
  '';
}
