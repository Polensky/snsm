{
  description = "Simple Notes - A tag-based note management system";

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
        pname = "simple-notes";
        version = "0.5.0";
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
