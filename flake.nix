{
  description = "snfsm - A simple tag-based note management system";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
        pname = "snsm";
        version = "0.9.0";
      in {
        packages.default = pkgs.buildGoModule {
          inherit pname version;
          src = ./.;
          vendorHash = "sha256-AYXefT2vl+uyoGMTQqmjuKrST8vgZjSyPnNQPxAapSs=";
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go compiler and tools
            go
            gopls
            gotools
            go-outline
          ];
        };
      }
    );
}
