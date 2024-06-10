{
  description = "flutties";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
  };

  outputs = { self, nixpkgs }:
    let
      allSystems = [
        "x86_64-linux" # 64-bit Intel/AMD Linux
        "aarch64-linux" # 64-bit ARM Linux
        "x86_64-darwin" # 64-bit Intel macOS
        "aarch64-darwin" # 64-bit ARM macOS
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        inherit system;
        pkgs = import nixpkgs { inherit system; };
      });
    in
    {
      packages = forAllSystems ({ system, pkgs, ... }:
        rec {
          default = flutties;

          flutties = pkgs.stdenv.mkDerivation {
            name = "flutties";
            src = ./.;
            pwd = ./.;
            buildInputs = [ pkgs.templ pkgs.go ];
            buildPhase = ''
              export HOME=$(pwd)
              templ generate
              go build
            '';
            installPhase = ''
              mkdir -p $out/bin
              mv flutties $out/bin
            '';
          };
        });

      # `nix develop` provides a shell containing development tools.
      devShell = forAllSystems ({ system, pkgs }:
        pkgs.mkShell {
          buildInputs = with pkgs; [
            (golangci-lint.override { buildGoModule = buildGo122Module; })
            cosign # Used to sign container images.
            go_1_22
            gopls
            goreleaser
            gotestsum
            ko # Used to build Docker images.
            templ
          ];
        });

      # This flake outputs an overlay that can be used to add templ and
      # templ-docs to nixpkgs as per https://templ.guide/quick-start/installation/#nix
      #
      # Example usage:
      #
      # nixpkgs.overlays = [
      #   inputs.templ.overlays.default
      # ];
      overlays.default = final: prev: {
        flutties = self.packages.${final.stdenv.system}.flutties;
      };
    };
}

