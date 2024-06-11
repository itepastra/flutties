{
  description = "flutties";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
    templ = {
      url = "github:a-h/templ";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    gitignore = {
      url = "github:hercules-ci/gitignore.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, gitignore, gomod2nix, templ, ... }:
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

      packages = forAllSystems
        ({ system, pkgs, ... }:
          let
            buildGoApplication = gomod2nix.legacyPackages.${system}.buildGoApplication;
          in
          rec {
            default = flutties;

            flutties = buildGoApplication
              {
                name = "flutties";
                src = gitignore.lib.gitignoreSource ./.;
                go = pkgs.go;
                pwd = ./.;
                CGO_ENABLED = 0;
                flags = [ "-trimpath" ];
                ldflags = [
                  "-s"
                  "-w"
                  "-extldflags -static"
                ];
              };
          });

      # `nix develop` provides a shell containing development tools.
      devShell = forAllSystems ({ system, pkgs }:
        let
          templPkg = templ.packages.${system}.templ;
          gomod2nixPkg = gomod2nix.legacyPackages.${system}.gomod2nix;
        in
        pkgs.mkShell {
          buildInputs = with pkgs; [
            (golangci-lint.override { buildGoModule = buildGo122Module; })
            cosign # Used to sign container images.
            go_1_22
            gopls
            goreleaser
            gotestsum
            ko # Used to build Docker images.
            templPkg
            gomod2nixPkg
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

